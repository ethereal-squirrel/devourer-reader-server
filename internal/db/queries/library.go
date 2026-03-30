package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func GetLibraryByID(d *sql.DB, id int64) (*db.Library, error) {
	row := d.QueryRow(`SELECT id, name, path, type, metadata FROM Library WHERE id=?`, id)
	return scanLibrary(row)
}

func GetLibraryByPath(d *sql.DB, path string) (*db.Library, error) {
	row := d.QueryRow(`SELECT id, name, path, type, metadata FROM Library WHERE path=?`, path)
	return scanLibrary(row)
}

func ListLibraries(d *sql.DB) ([]*db.Library, error) {
	rows, err := d.Query(`SELECT id, name, path, type, metadata FROM Library`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libs []*db.Library
	for rows.Next() {
		l, err := scanLibrary(rows)
		if err != nil {
			return nil, err
		}
		libs = append(libs, l)
	}
	return libs, rows.Err()
}

func CreateLibrary(d *sql.DB, name, path, libType string, metadata json.RawMessage) (*db.Library, error) {
	res, err := d.Exec(
		`INSERT INTO Library (name, path, type, metadata) VALUES (?, ?, ?, ?)`,
		name, path, libType, string(metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("create library: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetLibraryByID(d, id)
}

func UpdateLibrary(d *sql.DB, id int64, name, path string, metadata json.RawMessage) error {
	_, err := d.Exec(
		`UPDATE Library SET name=?, path=?, metadata=? WHERE id=?`,
		name, path, string(metadata), id,
	)
	return err
}

func DeleteLibrary(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM Library WHERE id=?`, id)
	return err
}

func CountBookFiles(d *sql.DB, libraryID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM BookFile WHERE library_id=?`, libraryID).Scan(&count)
	return count, err
}

func CountMangaSeries(d *sql.DB, libraryID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM MangaSeries WHERE library_id=?`, libraryID).Scan(&count)
	return count, err
}

func scanLibrary(s rowScanner) (*db.Library, error) {
	var l db.Library
	var metadata string
	if err := s.Scan(&l.ID, &l.Name, &l.Path, &l.Type, &metadata); err != nil {
		return nil, err
	}
	l.Metadata = json.RawMessage(metadata)
	return &l, nil
}
