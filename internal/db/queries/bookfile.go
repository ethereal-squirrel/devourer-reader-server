package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/devourer/server/internal/db"
)

func GetBookFileByID(d *sql.DB, id int64) (*db.BookFile, error) {
	row := d.QueryRow(`SELECT id, title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags FROM BookFile WHERE id=?`, id)
	return scanBookFile(row)
}

func GetBookFileByPath(d *sql.DB, path string) (*db.BookFile, error) {
	row := d.QueryRow(`SELECT id, title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags FROM BookFile WHERE path=?`, path)
	return scanBookFile(row)
}

func ListBookFilesByLibrary(d *sql.DB, libraryID int64) ([]*db.BookFile, error) {
	rows, err := d.Query(`SELECT id, title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags FROM BookFile WHERE library_id=? ORDER BY title`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookFiles(rows)
}

func SearchBookFiles(d *sql.DB, query string) ([]*db.BookFile, error) {
	rows, err := d.Query(`SELECT id, title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags FROM BookFile WHERE title LIKE ? OR file_name LIKE ? LIMIT 50`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookFiles(rows)
}

func ListBookFilesPreview(d *sql.DB, libraryID int64, limit int) ([]*db.BookFile, error) {
	rows, err := d.Query(`SELECT id, title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags FROM BookFile WHERE library_id=? LIMIT ?`, libraryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookFiles(rows)
}

func CreateBookFile(d *sql.DB, bf *db.BookFile) (*db.BookFile, error) {
	res, err := d.Exec(
		`INSERT INTO BookFile (title, path, file_name, file_format, total_pages, current_page, is_read, library_id, metadata, formats, tags) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bf.Title, bf.Path, bf.FileName, bf.FileFormat, bf.TotalPages, bf.CurrentPage, bf.IsRead,
		bf.LibraryID, string(bf.Metadata), string(bf.Formats), string(bf.Tags),
	)
	if err != nil {
		return nil, fmt.Errorf("create book file: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetBookFileByID(d, id)
}

func UpdateBookFileTotalPages(d *sql.DB, id int64, totalPages int) error {
	_, err := d.Exec(`UPDATE BookFile SET total_pages=? WHERE id=?`, totalPages, id)
	return err
}

func UpdateBookFileMetadata(d *sql.DB, id int64, metadata json.RawMessage) error {
	_, err := d.Exec(`UPDATE BookFile SET metadata=? WHERE id=?`, string(metadata), id)
	return err
}

func UpdateBookFileTags(d *sql.DB, id int64, tags json.RawMessage) error {
	_, err := d.Exec(`UPDATE BookFile SET tags=? WHERE id=?`, string(tags), id)
	return err
}

func DeleteBookFile(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM BookFile WHERE id=?`, id)
	return err
}

func DeleteBookFilesByLibrary(d *sql.DB, libraryID int64) error {
	_, err := d.Exec(`DELETE FROM BookFile WHERE library_id=?`, libraryID)
	return err
}

func ListBookFileIDsByLibrary(d *sql.DB, libraryID int64) ([]int64, error) {
	rows, err := d.Query(`SELECT id FROM BookFile WHERE library_id=?`, libraryID)
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

func scanBookFile(s rowScanner) (*db.BookFile, error) {
	var bf db.BookFile
	var metadata, formats, tags string
	if err := s.Scan(&bf.ID, &bf.Title, &bf.Path, &bf.FileName, &bf.FileFormat,
		&bf.TotalPages, &bf.CurrentPage, &bf.IsRead, &bf.LibraryID,
		&metadata, &formats, &tags); err != nil {
		return nil, err
	}
	bf.Metadata = json.RawMessage(metadata)
	bf.Formats = json.RawMessage(formats)
	bf.Tags = json.RawMessage(tags)
	return &bf, nil
}

func collectBookFiles(rows *sql.Rows) ([]*db.BookFile, error) {
	var files []*db.BookFile
	for rows.Next() {
		f, err := scanBookFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}
