package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devourer/server/internal/db"
)

func GetAudiobookFileByID(d *sql.DB, id int64) (*db.AudiobookFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata FROM AudiobookFile WHERE id=?`, id)
	return scanAudiobookFile(row)
}

func GetAudiobookFileByPath(d *sql.DB, path string) (*db.AudiobookFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata FROM AudiobookFile WHERE path=?`, path)
	return scanAudiobookFile(row)
}

func GetAudiobookFileBySeriesAndID(d *sql.DB, seriesID, id int64) (*db.AudiobookFile, error) {
	row := d.QueryRow(`SELECT id, path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata FROM AudiobookFile WHERE series_id=? AND id=?`, seriesID, id)
	return scanAudiobookFile(row)
}

func ListAudiobookFilesBySeries(d *sql.DB, seriesID int64) ([]*db.AudiobookFile, error) {
	rows, err := d.Query(`SELECT id, path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata FROM AudiobookFile WHERE series_id=? ORDER BY track_number ASC, file_name ASC`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudiobookFiles(rows)
}

func ListAudiobookFilesPathBySeries(d *sql.DB, seriesID int64) ([]*db.AudiobookFile, error) {
	rows, err := d.Query(`SELECT id, path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata FROM AudiobookFile WHERE series_id=?`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudiobookFiles(rows)
}

func CreateAudiobookFile(d *sql.DB, af *db.AudiobookFile) (*db.AudiobookFile, error) {
	res, err := d.Exec(
		`INSERT INTO AudiobookFile (path, file_name, file_format, track_number, duration_seconds, current_position_seconds, is_listened, series_id, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		af.Path, af.FileName, af.FileFormat, af.TrackNumber, af.DurationSeconds,
		af.CurrentPositionSeconds, af.IsListened, af.SeriesID, string(af.Metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("create audiobook file: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetAudiobookFileByID(d, id)
}

func DeleteAudiobookFile(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM AudiobookFile WHERE id=?`, id)
	return err
}

func DeleteAudiobookFilesBySeries(d *sql.DB, seriesID int64) error {
	_, err := d.Exec(`DELETE FROM AudiobookFile WHERE series_id=?`, seriesID)
	return err
}

func DeleteAudiobookFilesBySeriesIDs(d *sql.DB, seriesIDs []int64) error {
	if len(seriesIDs) == 0 {
		return nil
	}
	ph := strings.Repeat(",?", len(seriesIDs))[1:]
	args := make([]any, len(seriesIDs))
	for i, id := range seriesIDs {
		args[i] = id
	}
	_, err := d.Exec(fmt.Sprintf(`DELETE FROM AudiobookFile WHERE series_id IN (%s)`, ph), args...)
	return err
}

func scanAudiobookFile(s rowScanner) (*db.AudiobookFile, error) {
	var af db.AudiobookFile
	var metadata string
	if err := s.Scan(&af.ID, &af.Path, &af.FileName, &af.FileFormat,
		&af.TrackNumber, &af.DurationSeconds, &af.CurrentPositionSeconds,
		&af.IsListened, &af.SeriesID, &metadata); err != nil {
		return nil, err
	}
	af.Metadata = json.RawMessage(metadata)
	return &af, nil
}

func collectAudiobookFiles(rows *sql.Rows) ([]*db.AudiobookFile, error) {
	var files []*db.AudiobookFile
	for rows.Next() {
		f, err := scanAudiobookFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}
