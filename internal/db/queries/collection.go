package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func ListCollectionsByLibrary(d *sql.DB, libraryID, userID int64) ([]*db.Collection, error) {
	rows, err := d.Query(`SELECT id, library_id, name, series, user_id FROM Collection WHERE library_id=? AND user_id=?`, libraryID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCollections(rows)
}

func ListCollectionsByLibraryPublicOrUser(d *sql.DB, libraryID, userID int64) ([]*db.Collection, error) {
	rows, err := d.Query(`SELECT id, library_id, name, series, user_id FROM Collection WHERE library_id=? AND (user_id=? OR user_id=0)`, libraryID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCollections(rows)
}

func GetCollectionByID(d *sql.DB, id, userID int64) (*db.Collection, error) {
	row := d.QueryRow(`SELECT id, library_id, name, series, user_id FROM Collection WHERE id=? AND (user_id=? OR user_id=0)`, id, userID)
	return scanCollection(row)
}

func GetCollectionByLibraryNameAndUserPublic(d *sql.DB, libraryID int64, name string) (*db.Collection, error) {
	row := d.QueryRow(`SELECT id, library_id, name, series, user_id FROM Collection WHERE library_id=? AND name=? AND user_id=0`, libraryID, name)
	return scanCollection(row)
}

func CreateCollection(d *sql.DB, libraryID, userID int64, name string, series json.RawMessage) (*db.Collection, error) {
	res, err := d.Exec(
		`INSERT INTO Collection (library_id, name, series, user_id) VALUES (?, ?, ?, ?)`,
		libraryID, name, string(series), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}
	id, _ := res.LastInsertId()
	row := d.QueryRow(`SELECT id, library_id, name, series, user_id FROM Collection WHERE id=?`, id)
	return scanCollection(row)
}

func UpdateCollectionSeries(d *sql.DB, id int64, series json.RawMessage) error {
	_, err := d.Exec(`UPDATE Collection SET series=? WHERE id=?`, string(series), id)
	return err
}

func DeleteCollection(d *sql.DB, id, userID int64) error {
	_, err := d.Exec(`DELETE FROM Collection WHERE id=? AND user_id=?`, id, userID)
	return err
}

func DeleteCollectionsByLibrary(d *sql.DB, libraryID int64) error {
	_, err := d.Exec(`DELETE FROM Collection WHERE library_id=?`, libraryID)
	return err
}

func DeleteCollectionsByUserID(d *sql.DB, userID int64) error {
	_, err := d.Exec(`DELETE FROM Collection WHERE user_id=?`, userID)
	return err
}

func CountCollectionsByUser(d *sql.DB, userID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM Collection WHERE user_id=?`, userID).Scan(&count)
	return count, err
}

func scanCollection(s rowScanner) (*db.Collection, error) {
	var c db.Collection
	var series string
	if err := s.Scan(&c.ID, &c.LibraryID, &c.Name, &series, &c.UserID); err != nil {
		return nil, err
	}
	c.Series = json.RawMessage(series)
	return &c, nil
}

func collectCollections(rows *sql.Rows) ([]*db.Collection, error) {
	var cols []*db.Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}
