package queries

import (
	"database/sql"

	"github.com/devourer/server/internal/db"
)

func GetReadingStatus(d *sql.DB, userID, fileID int64, fileType string) (*db.ReadingStatus, error) {
	row := d.QueryRow(`SELECT id, user_id, file_type, file_id, current_page FROM ReadingStatus WHERE user_id=? AND file_id=? AND file_type=?`, userID, fileID, fileType)
	var rs db.ReadingStatus
	if err := row.Scan(&rs.ID, &rs.UserID, &rs.FileType, &rs.FileID, &rs.CurrentPage); err != nil {
		return nil, err
	}
	return &rs, nil
}

func UpsertReadingStatus(d *sql.DB, userID, fileID int64, fileType, currentPage string) error {
	existing, err := GetReadingStatus(d, userID, fileID, fileType)
	if err != nil {
		_, err = d.Exec(
			`INSERT INTO ReadingStatus (user_id, file_type, file_id, current_page) VALUES (?, ?, ?, ?)`,
			userID, fileType, fileID, currentPage,
		)
		return err
	}
	_, err = d.Exec(`UPDATE ReadingStatus SET current_page=? WHERE id=?`, currentPage, existing.ID)
	return err
}

func DeleteReadingStatus(d *sql.DB, userID, fileID int64, fileType string) error {
	_, err := d.Exec(`DELETE FROM ReadingStatus WHERE user_id=? AND file_id=? AND file_type=?`, userID, fileID, fileType)
	return err
}

func DeleteReadingStatusByFileID(d *sql.DB, fileID int64) error {
	_, err := d.Exec(`DELETE FROM ReadingStatus WHERE file_id=?`, fileID)
	return err
}

func DeleteReadingStatusByUserID(d *sql.DB, userID int64) error {
	_, err := d.Exec(`DELETE FROM ReadingStatus WHERE user_id=?`, userID)
	return err
}

func DeleteReadingStatusByFileType(d *sql.DB, fileType string, fileIDs []int64) error {
	if len(fileIDs) == 0 {
		return nil
	}
	args := make([]any, 0, len(fileIDs)+1)
	args = append(args, fileType)
	ph := "?"
	for i, id := range fileIDs {
		if i == 0 {
			args = append(args, id)
		} else {
			ph += ",?"
			args = append(args, id)
		}
	}
	_, err := d.Exec("DELETE FROM ReadingStatus WHERE file_type=? AND file_id IN ("+ph+")", args...)
	return err
}
