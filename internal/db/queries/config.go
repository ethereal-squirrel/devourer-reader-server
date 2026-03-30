package queries

import (
	"database/sql"

	"github.com/devourer/server/internal/db"
)

func GetConfig(d *sql.DB, key string) (*db.Config, error) {
	row := d.QueryRow(`SELECT id, key, value FROM Config WHERE key=?`, key)
	var c db.Config
	if err := row.Scan(&c.ID, &c.Key, &c.Value); err != nil {
		return nil, err
	}
	return &c, nil
}

func UpsertConfig(d *sql.DB, key, value string) error {
	_, err := d.Exec(
		`INSERT INTO Config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

func GetAllConfig(d *sql.DB) (map[string]string, error) {
	rows, err := d.Query(`SELECT key, value FROM Config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}
