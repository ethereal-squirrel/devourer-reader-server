package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func GetMangaSeriesByID(d *sql.DB, id int64) (*db.MangaSeries, error) {
	row := d.QueryRow(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE id=?`, id)
	return scanMangaSeries(row)
}

func GetMangaSeriesByPath(d *sql.DB, path string) (*db.MangaSeries, error) {
	row := d.QueryRow(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE path=?`, path)
	return scanMangaSeries(row)
}

func GetMangaSeriesByLibraryAndTitle(d *sql.DB, libraryID int64, title string) (*db.MangaSeries, error) {
	row := d.QueryRow(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE library_id=? AND title=?`, libraryID, title)
	return scanMangaSeries(row)
}

func GetMangaSeriesByLibraryAndID(d *sql.DB, libraryID, id int64) (*db.MangaSeries, error) {
	row := d.QueryRow(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE library_id=? AND id=?`, libraryID, id)
	return scanMangaSeries(row)
}

func ListMangaSeriesByLibrary(d *sql.DB, libraryID int64) ([]*db.MangaSeries, error) {
	rows, err := d.Query(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE library_id=? ORDER BY title`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMangaSeries(rows)
}

func ListAllMangaSeries(d *sql.DB) ([]*db.MangaSeries, error) {
	rows, err := d.Query(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMangaSeries(rows)
}

func ListMangaSeriesPreview(d *sql.DB, libraryID int64, limit int) ([]*db.MangaSeries, error) {
	rows, err := d.Query(`SELECT id, title, path, cover, library_id, manga_data FROM MangaSeries WHERE library_id=? LIMIT ?`, libraryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMangaSeries(rows)
}

func CreateMangaSeries(d *sql.DB, ms *db.MangaSeries) (*db.MangaSeries, error) {
	res, err := d.Exec(
		`INSERT INTO MangaSeries (title, path, cover, library_id, manga_data) VALUES (?, ?, ?, ?, ?)`,
		ms.Title, ms.Path, ms.Cover, ms.LibraryID, string(ms.MangaData),
	)
	if err != nil {
		return nil, fmt.Errorf("create manga series: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetMangaSeriesByID(d, id)
}

func UpdateMangaSeriesMetadata(d *sql.DB, id int64, mangaData json.RawMessage) error {
	_, err := d.Exec(`UPDATE MangaSeries SET manga_data=? WHERE id=?`, string(mangaData), id)
	return err
}

func DeleteMangaSeries(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM MangaSeries WHERE id=?`, id)
	return err
}

func DeleteMangaSeriesByLibrary(d *sql.DB, libraryID int64) error {
	_, err := d.Exec(`DELETE FROM MangaSeries WHERE library_id=?`, libraryID)
	return err
}

func ListMangaSeriesIDsByLibrary(d *sql.DB, libraryID int64) ([]int64, error) {
	rows, err := d.Query(`SELECT id FROM MangaSeries WHERE library_id=?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanMangaSeries(s rowScanner) (*db.MangaSeries, error) {
	var ms db.MangaSeries
	var mangaData string
	if err := s.Scan(&ms.ID, &ms.Title, &ms.Path, &ms.Cover, &ms.LibraryID, &mangaData); err != nil {
		return nil, err
	}
	ms.MangaData = json.RawMessage(mangaData)
	return &ms, nil
}

func collectMangaSeries(rows *sql.Rows) ([]*db.MangaSeries, error) {
	var series []*db.MangaSeries
	for rows.Next() {
		s, err := scanMangaSeries(rows)
		if err != nil {
			return nil, err
		}
		series = append(series, s)
	}
	return series, rows.Err()
}
