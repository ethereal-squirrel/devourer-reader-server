package migrate

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"

	_ "modernc.org/sqlite"
)

type calibreBook struct {
	ID         int64
	Title      string
	AuthorSort string
	Path       string
	ISBN       string
}

func MigrateCalibre(devourerDB *sql.DB, calibrePath, libraryName, metadataProvider string, userID int64) error {
	calibreDB := filepath.Join(calibrePath, "metadata.db")
	if _, err := os.Stat(calibreDB); os.IsNotExist(err) {
		return fmt.Errorf("calibre metadata.db not found at %s", calibrePath)
	}

	cdb, err := sql.Open("sqlite", calibreDB)
	if err != nil {
		return fmt.Errorf("open calibre DB: %w", err)
	}
	defer cdb.Close()

	metaJSON, _ := json.Marshal(map[string]any{
		"provider": metadataProvider,
	})
	lib, err := queries.CreateLibrary(devourerDB, libraryName, calibrePath, "book", json.RawMessage(metaJSON))
	if err != nil {
		return fmt.Errorf("create library: %w", err)
	}

	books, err := listCalibreBooks(cdb)
	if err != nil {
		return fmt.Errorf("list calibre books: %w", err)
	}

	log.Printf("[Calibre] Importing %d books into library %q", len(books), libraryName)

	for _, book := range books {
		if err := importCalibreBook(devourerDB, cdb, lib, calibrePath, book); err != nil {
			log.Printf("[Calibre] Skipping %q: %v", book.Title, err)
		}
	}

	log.Printf("[Calibre] Migration complete")
	return nil
}

func listCalibreBooks(cdb *sql.DB) ([]calibreBook, error) {
	rows, err := cdb.Query(`
		SELECT b.id, b.title, b.author_sort, b.path,
		       COALESCE((SELECT i.val FROM identifiers i WHERE i.book=b.id AND i.type='isbn' LIMIT 1), '') AS isbn
		FROM books b`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []calibreBook
	for rows.Next() {
		var b calibreBook
		if err := rows.Scan(&b.ID, &b.Title, &b.AuthorSort, &b.Path, &b.ISBN); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

var supportedFormats = map[string]bool{
	"EPUB": true, "PDF": true, "MOBI": true,
}

func importCalibreBook(devourerDB, cdb *sql.DB, lib *db.Library, calibrePath string, book calibreBook) error {
	rows, err := cdb.Query(`SELECT format, name FROM data WHERE book=?`, book.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type formatRow struct{ format, name string }
	var formats []formatRow
	for rows.Next() {
		var f formatRow
		if err := rows.Scan(&f.format, &f.name); err != nil {
			return err
		}
		formats = append(formats, f)
	}

	var chosenPath, chosenFormat, chosenName string
	for _, f := range formats {
		if !supportedFormats[f.format] {
			continue
		}
		ext := "." + f.format
		candidate := filepath.Join(calibrePath, book.Path, f.name+ext)
		if _, err := os.Stat(candidate); err == nil {
			chosenPath = candidate
			chosenFormat = f.format
			chosenName = f.name + ext
			break
		}
	}

	if chosenPath == "" {
		return fmt.Errorf("no supported format file found on disk")
	}

	if _, err := queries.GetBookFileByPath(devourerDB, chosenPath); err == nil {
		return nil
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"original_title": book.Title,
		"authors":        []string{book.AuthorSort},
		"isbn":           book.ISBN,
		"provider":       "calibre",
	})
	formatsJSON, _ := json.Marshal([]map[string]any{{
		"format": chosenFormat,
		"name":   chosenName,
		"path":   chosenPath,
	}})

	_, err = queries.CreateBookFile(devourerDB, &db.BookFile{
		Title:      book.Title,
		Path:       chosenPath,
		FileName:   chosenName,
		FileFormat: chosenFormat,
		LibraryID:  lib.ID,
		Metadata:   json.RawMessage(metaJSON),
		Formats:    json.RawMessage(formatsJSON),
		Tags:       json.RawMessage(`[]`),
	})
	return err
}
