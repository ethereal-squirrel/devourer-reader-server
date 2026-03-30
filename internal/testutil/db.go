package testutil

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const testJWTSecret = "test-jwt-secret-for-unit-tests"

const schema = `
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
CREATE UNIQUE INDEX "Library_name_key" ON "Library"("name");
CREATE UNIQUE INDEX "Library_path_key" ON "Library"("path");

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
    "tags" JSONB NOT NULL,
    CONSTRAINT "BookFile_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "BookFile_path_key" ON "BookFile"("path");

CREATE TABLE "MangaSeries" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "cover" TEXT NOT NULL,
    "library_id" INTEGER NOT NULL,
    "manga_data" JSONB NOT NULL,
    CONSTRAINT "MangaSeries_library_id_fkey" FOREIGN KEY ("library_id") REFERENCES "Library" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "MangaSeries_path_key" ON "MangaSeries"("path");

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
    "metadata" JSONB NOT NULL,
    CONSTRAINT "MangaFile_series_id_fkey" FOREIGN KEY ("series_id") REFERENCES "MangaSeries" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "MangaFile_path_key" ON "MangaFile"("path");

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

func NewDB(t testing.TB) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testutil.NewDB: open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	if _, err := d.Exec(schema); err != nil {
		t.Fatalf("testutil.NewDB: apply schema: %v", err)
	}

	_, err = d.Exec(`INSERT INTO Roles (title,is_admin,add_file,delete_file,edit_metadata,manage_collections,manage_library,create_user) VALUES
		('admin',    1,1,1,1,1,1,1),
		('moderator',0,1,1,1,1,0,0),
		('upload',   0,1,0,0,0,0,0),
		('user',     0,0,0,0,0,0,0)`)
	if err != nil {
		t.Fatalf("testutil.NewDB: seed roles: %v", err)
	}

	seeds := [][2]string{
		{"jwt_secret", testJWTSecret},
		{"allow_public", "0"},
		{"allow_register", "0"},
	}
	for _, s := range seeds {
		if _, err := d.Exec(`INSERT INTO Config (key, value) VALUES (?, ?)`, s[0], s[1]); err != nil {
			t.Fatalf("testutil.NewDB: seed config %q: %v", s[0], err)
		}
	}

	return d
}

func NewDBNoSecret(t testing.TB) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testutil.NewDBNoSecret: open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := d.Exec(schema); err != nil {
		t.Fatalf("testutil.NewDBNoSecret: apply schema: %v", err)
	}
	return d
}

func MustInsertUser(t testing.TB, d *sql.DB, email, hashedPw string, roles []string) int64 {
	t.Helper()
	rolesJSON, _ := json.Marshal(roles)
	defaultMeta := `{"settings":{}}`
	res, err := d.Exec(
		`INSERT INTO User (email, password, api_key, roles, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		email, hashedPw, "test-api-key", string(rolesJSON), defaultMeta,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("testutil.MustInsertUser(%q): %v", email, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func MustInsertLibrary(t testing.TB, d *sql.DB, name, path, libType string) int64 {
	t.Helper()
	res, err := d.Exec(
		`INSERT INTO Library (name, path, type, metadata) VALUES (?, ?, ?, '{}')`,
		name, path, libType,
	)
	if err != nil {
		t.Fatalf("testutil.MustInsertLibrary(%q): %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func JWTSecret() string { return testJWTSecret }

func FormatUserRoles(roles []string) string {
	b, _ := json.Marshal(roles)
	return fmt.Sprintf("%s", b)
}
