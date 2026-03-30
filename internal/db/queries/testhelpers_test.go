package queries

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const testSchema = `
CREATE TABLE "Config" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL
);
CREATE UNIQUE INDEX "Config_key_key" ON "Config"("key");

CREATE TABLE "User" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "email" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "api_key" TEXT NOT NULL,
    "roles" JSONB NOT NULL,
    "metadata" JSONB NOT NULL,
    "created_at" DATETIME NOT NULL
);
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

CREATE TABLE "Library" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "metadata" JSONB NOT NULL
);

CREATE TABLE "BookFile" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_format" TEXT NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL,
    "is_read" BOOLEAN NOT NULL,
    "library_id" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    "formats" JSONB NOT NULL,
    "tags" JSONB NOT NULL
);

CREATE TABLE "MangaSeries" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "cover" TEXT NOT NULL,
    "library_id" INTEGER NOT NULL,
    "manga_data" JSONB NOT NULL
);

CREATE TABLE "MangaFile" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "path" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_format" TEXT NOT NULL,
    "volume" INTEGER NOT NULL,
    "chapter" INTEGER NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "current_page" INTEGER NOT NULL,
    "is_read" BOOLEAN NOT NULL,
    "series_id" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL
);

CREATE TABLE "RecentlyRead" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "is_local" BOOLEAN NOT NULL,
    "library_id" INTEGER NOT NULL,
    "series_id" INTEGER NOT NULL,
    "file_id" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL,
    "total_pages" INTEGER NOT NULL,
    "volume" INTEGER NOT NULL,
    "chapter" INTEGER NOT NULL,
    "user_id" INTEGER NOT NULL
);

CREATE TABLE "Collection" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "library_id" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "series" JSONB NOT NULL,
    "user_id" INTEGER NOT NULL
);

CREATE TABLE "Roles" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "is_admin" BOOLEAN NOT NULL,
    "add_file" BOOLEAN NOT NULL,
    "delete_file" BOOLEAN NOT NULL,
    "edit_metadata" BOOLEAN NOT NULL,
    "manage_collections" BOOLEAN NOT NULL,
    "manage_library" BOOLEAN NOT NULL,
    "create_user" BOOLEAN NOT NULL
);

CREATE TABLE "ReadingStatus" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "current_page" TEXT NOT NULL
);

CREATE TABLE "UserRating" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "rating" INTEGER NOT NULL
);

CREATE TABLE "UserTag" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "file_type" TEXT NOT NULL,
    "file_id" INTEGER NOT NULL,
    "tag" TEXT NOT NULL
);
`

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("newTestDB: open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	if _, err := d.Exec(testSchema); err != nil {
		t.Fatalf("newTestDB: apply schema: %v", err)
	}

	seeds := [][2]string{
		{"jwt_secret", "test-secret"},
		{"allow_public", "0"},
		{"allow_register", "0"},
	}
	for _, s := range seeds {
		if _, err := d.Exec(`INSERT INTO Config (key, value) VALUES (?, ?)`, s[0], s[1]); err != nil {
			t.Fatalf("newTestDB: seed config %q: %v", s[0], err)
		}
	}

	return d
}

func insertUser(t *testing.T, d *sql.DB, email, hashedPw string, roles []string) int64 {
	t.Helper()
	rolesJSON, _ := json.Marshal(roles)
	res, err := d.Exec(
		`INSERT INTO User (email, password, api_key, roles, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		email, hashedPw, "test-api-key", string(rolesJSON), `{"settings":{}}`,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insertUser(%q): %v", email, err)
	}
	id, _ := res.LastInsertId()
	return id
}
