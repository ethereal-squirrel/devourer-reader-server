package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devourer/server/internal/db"
)

const audiobookSeriesCols = `id, title, path, cover, library_id, audiobook_data, total_duration_seconds`

func GetAudiobookSeriesByID(d *sql.DB, id int64) (*db.AudiobookSeries, error) {
	row := d.QueryRow(`SELECT `+audiobookSeriesCols+` FROM AudiobookSeries WHERE id=?`, id)
	return scanAudiobookSeries(row)
}

func GetAudiobookSeriesByPath(d *sql.DB, path string) (*db.AudiobookSeries, error) {
	row := d.QueryRow(`SELECT `+audiobookSeriesCols+` FROM AudiobookSeries WHERE path=?`, path)
	return scanAudiobookSeries(row)
}

func GetAudiobookSeriesByLibraryAndID(d *sql.DB, libraryID, id int64) (*db.AudiobookSeries, error) {
	row := d.QueryRow(`SELECT `+audiobookSeriesCols+` FROM AudiobookSeries WHERE library_id=? AND id=?`, libraryID, id)
	return scanAudiobookSeries(row)
}

func ListAudiobookSeriesByLibrary(d *sql.DB, libraryID int64) ([]*db.AudiobookSeries, error) {
	rows, err := d.Query(`SELECT `+audiobookSeriesCols+` FROM AudiobookSeries WHERE library_id=? ORDER BY title`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudiobookSeries(rows)
}

func ListAudiobookSeriesPreview(d *sql.DB, libraryID int64, limit int) ([]*db.AudiobookSeries, error) {
	rows, err := d.Query(`SELECT `+audiobookSeriesCols+` FROM AudiobookSeries WHERE library_id=? LIMIT ?`, libraryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudiobookSeries(rows)
}

func ListAudiobookSeriesIDsByLibrary(d *sql.DB, libraryID int64) ([]int64, error) {
	rows, err := d.Query(`SELECT id FROM AudiobookSeries WHERE library_id=?`, libraryID)
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

func CountAudiobookSeries(d *sql.DB, libraryID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM AudiobookSeries WHERE library_id=?`, libraryID).Scan(&count)
	return count, err
}

func CreateAudiobookSeries(d *sql.DB, as *db.AudiobookSeries) (*db.AudiobookSeries, error) {
	res, err := d.Exec(
		`INSERT INTO AudiobookSeries (title, path, cover, library_id, audiobook_data, total_duration_seconds) VALUES (?, ?, ?, ?, ?, ?)`,
		as.Title, as.Path, as.Cover, as.LibraryID, string(as.AudiobookData), as.TotalDurationSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("create audiobook series: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetAudiobookSeriesByID(d, id)
}

func UpdateAudiobookSeriesMetadata(d *sql.DB, id int64, audiobookData json.RawMessage) error {
	_, err := d.Exec(`UPDATE AudiobookSeries SET audiobook_data=? WHERE id=?`, string(audiobookData), id)
	return err
}

func UpdateAudiobookSeriesTotalDuration(d *sql.DB, id int64, totalDurationSeconds int) error {
	_, err := d.Exec(`UPDATE AudiobookSeries SET total_duration_seconds=? WHERE id=?`, totalDurationSeconds, id)
	return err
}

func DeleteAudiobookSeries(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM AudiobookSeries WHERE id=?`, id)
	return err
}

func DeleteAudiobookSeriesByLibrary(d *sql.DB, libraryID int64) error {
	_, err := d.Exec(`DELETE FROM AudiobookSeries WHERE library_id=?`, libraryID)
	return err
}

func scanAudiobookSeries(s rowScanner) (*db.AudiobookSeries, error) {
	var as db.AudiobookSeries
	var audiobookData string
	if err := s.Scan(&as.ID, &as.Title, &as.Path, &as.Cover, &as.LibraryID, &audiobookData, &as.TotalDurationSeconds); err != nil {
		return nil, err
	}
	as.AudiobookData = json.RawMessage(audiobookData)
	return &as, nil
}

func collectAudiobookSeries(rows *sql.Rows) ([]*db.AudiobookSeries, error) {
	var series []*db.AudiobookSeries
	for rows.Next() {
		s, err := scanAudiobookSeries(rows)
		if err != nil {
			return nil, err
		}
		series = append(series, s)
	}
	return series, rows.Err()
}
