package queries

import (
	"database/sql"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func ListRecentlyRead(d *sql.DB, userID int64, limit int) ([]*db.RecentlyRead, error) {
	rows, err := d.Query(`SELECT id, is_local, library_id, series_id, file_id, current_page, total_pages, volume, chapter, user_id FROM RecentlyRead WHERE user_id=? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*db.RecentlyRead
	for rows.Next() {
		var rr db.RecentlyRead
		if err := rows.Scan(&rr.ID, &rr.IsLocal, &rr.LibraryID, &rr.SeriesID, &rr.FileID, &rr.CurrentPage, &rr.TotalPages, &rr.Volume, &rr.Chapter, &rr.UserID); err != nil {
			return nil, err
		}
		result = append(result, &rr)
	}
	return result, rows.Err()
}

func DeleteRecentlyReadByLibraryAndFile(d *sql.DB, libraryID, fileID, userID int64) error {
	_, err := d.Exec(`DELETE FROM RecentlyRead WHERE library_id=? AND file_id=? AND user_id=?`, libraryID, fileID, userID)
	return err
}

func CreateRecentlyRead(d *sql.DB, rr *db.RecentlyRead) error {
	_, err := d.Exec(
		`INSERT INTO RecentlyRead (is_local, library_id, series_id, file_id, current_page, total_pages, volume, chapter, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rr.IsLocal, rr.LibraryID, rr.SeriesID, rr.FileID, rr.CurrentPage, rr.TotalPages, rr.Volume, rr.Chapter, rr.UserID,
	)
	return err
}

func TrimRecentlyRead(d *sql.DB, userID int64, limit int) error {
	_, err := d.Exec(`
		DELETE FROM RecentlyRead
		WHERE user_id = ?
		AND id NOT IN (
			SELECT id FROM RecentlyRead
			WHERE user_id = ?
			ORDER BY id DESC
			LIMIT ?
		)`, userID, userID, limit)
	return err
}

func DeleteRecentlyReadByLibrary(d *sql.DB, libraryID int64, fileIDs []int64) error {
	if len(fileIDs) == 0 {
		_, err := d.Exec(`DELETE FROM RecentlyRead WHERE library_id=?`, libraryID)
		return err
	}
	ph := "?"
	args := make([]any, 0, len(fileIDs)+1)
	args = append(args, libraryID, fileIDs[0])
	for _, id := range fileIDs[1:] {
		ph += ",?"
		args = append(args, id)
	}
	_, err := d.Exec(fmt.Sprintf("DELETE FROM RecentlyRead WHERE library_id=? AND file_id IN (%s)", ph), args...)
	return err
}

func DeleteRecentlyReadByFileID(d *sql.DB, fileID int64) error {
	_, err := d.Exec(`DELETE FROM RecentlyRead WHERE file_id=?`, fileID)
	return err
}

func DeleteRecentlyReadByUserID(d *sql.DB, userID int64) error {
	_, err := d.Exec(`DELETE FROM RecentlyRead WHERE user_id=?`, userID)
	return err
}
