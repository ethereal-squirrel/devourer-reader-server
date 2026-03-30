package queries

import (
	"database/sql"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func ListUserTags(d *sql.DB, userID, fileID int64, fileType string) ([]*db.UserTag, error) {
	rows, err := d.Query(`SELECT id, user_id, file_type, file_id, tag FROM UserTag WHERE user_id=? AND file_id=? AND file_type=?`, userID, fileID, fileType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*db.UserTag
	for rows.Next() {
		var t db.UserTag
		if err := rows.Scan(&t.ID, &t.UserID, &t.FileType, &t.FileID, &t.Tag); err != nil {
			return nil, err
		}
		tags = append(tags, &t)
	}
	return tags, rows.Err()
}

func CreateUserTag(d *sql.DB, userID, fileID int64, fileType, tag string) error {
	_, err := d.Exec(`INSERT INTO UserTag (user_id, file_type, file_id, tag) VALUES (?, ?, ?, ?)`, userID, fileType, fileID, tag)
	return err
}

func DeleteUserTag(d *sql.DB, userID, fileID int64, fileType, tag string) error {
	_, err := d.Exec(`DELETE FROM UserTag WHERE user_id=? AND file_id=? AND file_type=? AND tag=?`, userID, fileID, fileType, tag)
	return err
}

func DeleteUserTagsByFileID(d *sql.DB, fileID int64) error {
	_, err := d.Exec(`DELETE FROM UserTag WHERE file_id=?`, fileID)
	return err
}

func DeleteUserTagsByFileType(d *sql.DB, fileType string, fileIDs []int64) error {
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
	_, err := d.Exec(fmt.Sprintf("DELETE FROM UserTag WHERE file_type=? AND file_id IN (%s)", ph), args...)
	return err
}
