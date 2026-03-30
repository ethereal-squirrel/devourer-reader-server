package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

//go:embed migrations/1.sql
var migration1 string

//go:embed migrations/2.sql
var migration2 string

//go:embed migrations/3.sql
var migration3 string

//go:embed migrations/4.sql
var migration4 string

//go:embed migrations/5.sql
var migration5 string

//go:embed migrations/6.sql
var migration6 string

//go:embed migrations/7.sql
var migration7 string

//go:embed migrations/8.sql
var migration8 string

var migrations = map[int]string{
	1: migration1,
	2: migration2,
	3: migration3,
	4: migration4,
	5: migration5,
	6: migration6,
	7: migration7,
	8: migration8,
}

const DatabaseVersion = 8

func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	db.Exec(`PRAGMA synchronous=NORMAL`)
	db.Exec(`PRAGMA cache_size=-64000`)
	db.Exec(`PRAGMA temp_store=MEMORY`)

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

func Initialize(db *sql.DB) error {
	var tableName string
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='Config'`)
	if err := row.Scan(&tableName); err != nil {
		log.Println("[DB] Tables do not exist, running initial migration...")
		if err := runMigration(db, 1, migrations[1]); err != nil {
			return fmt.Errorf("migration 1: %w", err)
		}

		if err := seedInitialData(db); err != nil {
			return fmt.Errorf("seed initial data: %w", err)
		}

		if err := setConfigValue(db, "migration_version", "1"); err != nil {
			return err
		}
	}

	currentVersion := 0
	var versionStr string
	row = db.QueryRow(`SELECT value FROM Config WHERE key='migration_version'`)
	if err := row.Scan(&versionStr); err == nil {
		fmt.Sscanf(versionStr, "%d", &currentVersion)
	}

	if currentVersion < DatabaseVersion {
		log.Printf("[DB] Database is at version %d, upgrading to %d...", currentVersion, DatabaseVersion)
		for i := currentVersion + 1; i <= DatabaseVersion; i++ {
			sql, ok := migrations[i]
			if !ok {
				continue
			}
			log.Printf("[DB] Running migration %d...", i)
			if err := runMigration(db, i, sql); err != nil {
				return fmt.Errorf("migration %d: %w", i, err)
			}
		}

		if err := setConfigValue(db, "migration_version", fmt.Sprintf("%d", DatabaseVersion)); err != nil {
			return err
		}
		log.Println("[DB] Migrations complete.")
	}

	log.Println("[DB] Database initialized successfully.")
	return nil
}

func runMigration(db *sql.DB, version int, content string) error {
	statements := strings.Split(content, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("exec %q: %w", stmt[:min(50, len(stmt))], err)
			}
		}
	}
	return nil
}

func seedInitialData(db *sql.DB) error {
	configs := []struct{ k, v string }{
		{"allow_public", "0"},
		{"allow_register", "0"},
		{"api_google_books", ""},
		{"jwt_secret", uuid.New().String() + uuid.New().String()},
	}
	for _, c := range configs {
		if _, err := db.Exec(`INSERT INTO Config (key, value) VALUES (?, ?)`, c.k, c.v); err != nil {
			return fmt.Errorf("insert config %s: %w", c.k, err)
		}
	}

	randomPassword := randomString(12)
	randomApiKey := uuid.New().String()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), 12)
	if err != nil {
		return err
	}
	hashedApiKey, err := bcrypt.GenerateFromPassword([]byte(randomApiKey), 12)
	if err != nil {
		return err
	}

	defaultSettings := `{"settings":{"book_pagemode":"single","book_font":"default","book_background":"#000000","manga_direction":"ltr","manga_pagemode":"single","manga_resizemode":"fit","manga_background":"#000000"}}`

	_, err = db.Exec(
		`INSERT INTO User (email, password, api_key, roles, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"admin", string(hashedPassword), string(hashedApiKey),
		`["admin"]`, defaultSettings,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	log.Println("[DB] --------------------------------")
	log.Println("[DB] Initial account created:")
	log.Println("[DB] Username: admin")
	log.Printf("[DB] Password: %s\n", randomPassword)
	log.Printf("[DB] API key:  %s\n", randomApiKey)
	log.Println("[DB] --------------------------------")

	return nil
}

func setConfigValue(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO Config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func GetConfigValue(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM Config WHERE key=?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func SetConfigValue(db *sql.DB, key, value string) error {
	return setConfigValue(db, key, value)
}

func randomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
