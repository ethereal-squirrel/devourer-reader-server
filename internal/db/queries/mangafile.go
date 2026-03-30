package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devourer/server/internal/db"
)

func GetMangaFileByID(d *sql.DB, id int64) (*db.MangaFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE id=?`, id)
	return scanMangaFile(row)
}

func GetMangaFileByPath(d *sql.DB, path string) (*db.MangaFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE path=?`, path)
	return scanMangaFile(row)
}

func GetMangaFileBySeriesAndID(d *sql.DB, seriesID, id int64) (*db.MangaFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE series_id=? AND id=?`, seriesID, id)
	return scanMangaFile(row)
}

func GetMangaFileBySeriesAndVolume(d *sql.DB, seriesID int64, volume int) (*db.MangaFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE series_id=? AND volume=? LIMIT 1`, seriesID, volume)
	return scanMangaFile(row)
}

func GetMangaFileBySeriesAndChapter(d *sql.DB, seriesID int64, chapter int) (*db.MangaFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE series_id=? AND chapter=? LIMIT 1`, seriesID, chapter)
	return scanMangaFile(row)
}

func ListMangaFilesBySeries(d *sql.DB, seriesID int64) ([]*db.MangaFile, error) {
	rows, err := d.Query(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE series_id=? ORDER BY volume ASC, chapter ASC`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMangaFiles(rows)
}

func ListMangaFilesPathBySeries(d *sql.DB, seriesID int64) ([]*db.MangaFile, error) {
	rows, err := d.Query(`SELECT id, path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata FROM MangaFile WHERE series_id=?`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMangaFiles(rows)
}

func CreateMangaFile(d *sql.DB, mf *db.MangaFile) (*db.MangaFile, error) {
	res, err := d.Exec(
		`INSERT INTO MangaFile (path, file_name, file_format, volume, chapter, total_pages, current_page, is_read, series_id, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mf.Path, mf.FileName, mf.FileFormat, mf.Volume, mf.Chapter,
		mf.TotalPages, mf.CurrentPage, mf.IsRead, mf.SeriesID, string(mf.Metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("create manga file: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetMangaFileByID(d, id)
}

func DeleteMangaFile(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM MangaFile WHERE id=?`, id)
	return err
}

func DeleteMangaFilesBySeries(d *sql.DB, seriesID int64) error {
	_, err := d.Exec(`DELETE FROM MangaFile WHERE series_id=?`, seriesID)
	return err
}

func DeleteMangaFilesBySeriesIDs(d *sql.DB, seriesIDs []int64) error {
	if len(seriesIDs) == 0 {
		return nil
	}
	ph := strings.Repeat(",?", len(seriesIDs))[1:]
	args := make([]any, len(seriesIDs))
	for i, id := range seriesIDs {
		args[i] = id
	}
	_, err := d.Exec(fmt.Sprintf(`DELETE FROM MangaFile WHERE series_id IN (%s)`, ph), args...)
	return err
}

func scanMangaFile(s rowScanner) (*db.MangaFile, error) {
	var mf db.MangaFile
	var metadata string
	if err := s.Scan(&mf.ID, &mf.Path, &mf.FileName, &mf.FileFormat,
		&mf.Volume, &mf.Chapter, &mf.TotalPages, &mf.CurrentPage,
		&mf.IsRead, &mf.SeriesID, &metadata); err != nil {
		return nil, err
	}
	mf.Metadata = json.RawMessage(metadata)
	return &mf, nil
}

func collectMangaFiles(rows *sql.Rows) ([]*db.MangaFile, error) {
	var files []*db.MangaFile
	for rows.Next() {
		f, err := scanMangaFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}
