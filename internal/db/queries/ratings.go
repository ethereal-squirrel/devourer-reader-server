package queries

import (
	"database/sql"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func GetUserRating(d *sql.DB, userID, fileID int64, fileType string) (*db.UserRating, error) {
	row := d.QueryRow(`SELECT id, user_id, file_type, file_id, rating FROM UserRating WHERE user_id=? AND file_id=? AND file_type=?`, userID, fileID, fileType)
	var r db.UserRating
	if err := row.Scan(&r.ID, &r.UserID, &r.FileType, &r.FileID, &r.Rating); err != nil {
		return nil, err
	}
	return &r, nil
}

func ListUserRatingsByLibraryAndType(d *sql.DB, userID int64, fileType string, fileIDs []int64) ([]*db.UserRating, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	ph := "?"
	args := make([]any, 0, len(fileIDs)+2)
	args = append(args, userID, fileType, fileIDs[0])
	for _, id := range fileIDs[1:] {
		ph += ",?"
		args = append(args, id)
	}
	rows, err := d.Query(fmt.Sprintf("SELECT id, user_id, file_type, file_id, rating FROM UserRating WHERE user_id=? AND file_type=? AND file_id IN (%s)", ph), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ratings []*db.UserRating
	for rows.Next() {
		var r db.UserRating
		if err := rows.Scan(&r.ID, &r.UserID, &r.FileType, &r.FileID, &r.Rating); err != nil {
			return nil, err
		}
		ratings = append(ratings, &r)
	}
	return ratings, rows.Err()
}

func UpsertUserRating(d *sql.DB, userID, fileID int64, fileType string, rating int) error {
	existing, err := GetUserRating(d, userID, fileID, fileType)
	if err != nil {
		_, err = d.Exec(`INSERT INTO UserRating (user_id, file_type, file_id, rating) VALUES (?, ?, ?, ?)`, userID, fileType, fileID, rating)
		return err
	}
	_, err = d.Exec(`UPDATE UserRating SET rating=? WHERE id=?`, rating, existing.ID)
	return err
}

func DeleteUserRatingsByFileID(d *sql.DB, fileID int64) error {
	_, err := d.Exec(`DELETE FROM UserRating WHERE file_id=?`, fileID)
	return err
}

func DeleteUserRatingsByFileType(d *sql.DB, fileType string, fileIDs []int64) error {
	if len(fileIDs) == 0 {
		return nil
	}
	ph := "?"
	args := make([]any, 0, len(fileIDs)+1)
	args = append(args, fileType, fileIDs[0])
	for _, id := range fileIDs[1:] {
		ph += ",?"
		args = append(args, id)
	}
	_, err := d.Exec(fmt.Sprintf("DELETE FROM UserRating WHERE file_type=? AND file_id IN (%s)", ph), args...)
	return err
}
