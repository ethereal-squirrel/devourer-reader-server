package scanner

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	imgconvert "github.com/devourer/server/internal/image"
	"github.com/devourer/server/internal/metadata"
)

var validBookExts = map[string]bool{
	".epub": true, ".pdf": true, ".mobi": true, ".txt": true,
	".docx": true, ".doc": true, ".rtf": true, ".html": true,
}

func isValidBookFile(path string) bool {
	return validBookExts[strings.ToLower(filepath.Ext(path))]
}

func ProcessBook(cfg *Config, lib *db.Library, filePath string) (*db.BookFile, error) {
	if _, err := queries.GetBookFileByPath(cfg.DB, filePath); err == nil {
		return nil, nil
	}

	baseName := filepath.Base(filePath)
	clean := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	clean = strings.TrimSpace(strings.NewReplacer(
		"[", " ", "]", " ", "(", " ", ")", " ", "<", " ", ">", " ",
	).Replace(clean))

	var epubMeta *EpubMetadata
	if strings.HasSuffix(strings.ToLower(filePath), ".epub") {
		epubMeta, _ = ScanEpub(filePath)
	}

	title := clean
	if epubMeta != nil && epubMeta.Title != "" {
		title = epubMeta.Title
	}

	providerKey := "googlebooks"
	if lib.Metadata != nil {
		var libMeta map[string]any
		if json.Unmarshal(lib.Metadata, &libMeta) == nil {
			if p, ok := libMeta["provider"].(string); ok && p != "" {
				providerKey = p
			}
		}
	}

	remoteMeta := fetchBookMetadata(cfg.Providers, providerKey, title, func() string {
		if epubMeta != nil {
			return epubMeta.ISBN
		}
		return ""
	}())

	combinedMeta := make(map[string]any)
	if remoteMeta != nil {
		for k, v := range remoteMeta {
			combinedMeta[k] = v
		}
	}
	if combinedMeta["title"] == nil {
		combinedMeta["title"] = title
	}
	if epubMeta != nil {
		epubSnapshot := map[string]any{
			"title":       epubMeta.Title,
			"author":      epubMeta.Author,
			"publisher":   epubMeta.Publisher,
			"date":        epubMeta.Date,
			"description": epubMeta.Description,
			"language":    epubMeta.Language,
			"isbn":        epubMeta.ISBN,
			"cover":       nil,
		}
		combinedMeta["epub"] = epubSnapshot
	}

	metaJSON, _ := json.Marshal(combinedMeta)
	formatsJSON, _ := json.Marshal([]map[string]any{{
		"format": strings.TrimPrefix(filepath.Ext(filePath), "."),
		"name":   baseName,
		"path":   filePath,
	}})

	pageCount := 0

	titleStr, _ := combinedMeta["title"].(string)
	file, err := queries.CreateBookFile(cfg.DB, &db.BookFile{
		Title:      titleStr,
		Path:       filePath,
		FileName:   baseName,
		FileFormat: strings.TrimPrefix(filepath.Ext(filePath), "."),
		TotalPages: pageCount,
		LibraryID:  lib.ID,
		Metadata:   json.RawMessage(metaJSON),
		Formats:    json.RawMessage(formatsJSON),
		Tags:       json.RawMessage(`[]`),
	})
	if err != nil {
		return nil, fmt.Errorf("create book file: %w", err)
	}

	coverDir := filepath.Join(lib.MetaBase(), "files", fmt.Sprintf("%d", file.ID))
	os.MkdirAll(coverDir, 0o755)
	coverPath := filepath.Join(coverDir, "cover.jpg")

	saved := false

	if epubMeta != nil && len(epubMeta.CoverData) > 0 {
		if err := imgconvert.ResizeAndSave(epubMeta.CoverData, coverPath, imgconvert.CoverMaxWidth); err == nil {
			saved = true
		}
	}

	if !saved && strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		if data, err := ProcessPDF(filePath); err == nil && len(data) > 0 {
			if err := imgconvert.ResizeAndSave(data, coverPath, imgconvert.CoverMaxWidth); err == nil {
				saved = true
			}
		}
	}

	if !saved {
		coverURL := ""
		if u, ok := combinedMeta["cover"].(string); ok && len(u) > 10 {
			coverURL = u
		} else if isbn, ok := combinedMeta["isbn_13"].(string); ok && len(isbn) > 0 {
			coverURL = "https://covers.openlibrary.org/b/isbn/" + isbn + "-L.jpg"
		}
		if coverURL != "" {
			imgconvert.DownloadAndSave(coverURL, coverPath, imgconvert.CoverMaxWidth)
		}
	}

	log.Printf("[Scanner] Created book: %s | %s", file.Title, file.Path)
	return file, nil
}

func ScanBookLibrary(cfg *Config, lib *db.Library, topLevel []string) {
	type collection struct {
		folder   string
		contents []int64
	}
	collections := make(map[string]*collection)

	for _, entry := range topLevel {
		fullPath := filepath.Join(lib.Path, entry)
		fi, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if !fi.IsDir() {
			if !isValidBookFile(fullPath) {
				log.Printf("[Scanner] Skipping non-book file: %s", entry)
				updateSeriesStatus(lib.ID, entry, "complete")
				continue
			}
			_, err := ProcessBook(cfg, lib, fullPath)
			if err != nil {
				updateSeriesError(lib.ID, entry, err.Error())
			} else {
				updateSeriesStatus(lib.ID, entry, "complete")
			}
			continue
		}

		files := getAllFiles(fullPath)
		var createdIDs []int64
		for _, f := range files {
			if !isValidBookFile(f) {
				continue
			}
			created, err := ProcessBook(cfg, lib, f)
			if err != nil {
				log.Printf("[Scanner] Error processing %s: %v", f, err)
				continue
			}
			if created != nil {
				createdIDs = append(createdIDs, created.ID)
			}
		}
		if len(createdIDs) > 1 {
			collections[entry] = &collection{folder: entry, contents: createdIDs}
		}
		updateSeriesStatus(lib.ID, entry, "complete")
	}

	for _, c := range collections {
		existing, err := queries.GetCollectionByLibraryNameAndUserPublic(cfg.DB, lib.ID, c.folder)
		if err != nil {
			contentsJSON, _ := json.Marshal(c.contents)
			queries.CreateCollection(cfg.DB, lib.ID, 0, c.folder, json.RawMessage(contentsJSON))
			continue
		}

		var existingIDs []int64
		json.Unmarshal(existing.Series, &existingIDs)
		seen := make(map[int64]bool)
		for _, id := range existingIDs {
			seen[id] = true
		}
		for _, id := range c.contents {
			if !seen[id] {
				existingIDs = append(existingIDs, id)
			}
		}
		merged, _ := json.Marshal(existingIDs)
		queries.UpdateCollectionSeries(cfg.DB, existing.ID, json.RawMessage(merged))
	}

	log.Printf("[Scanner] Book scan complete for library %d", lib.ID)
}

func fetchBookMetadata(providers map[string]*metadata.Provider, providerKey, title, isbn string) map[string]any {
	if providers == nil {
		return nil
	}

	by := "title"
	query := title

	if isbn != "" {
		switch len(isbn) {
		case 13:
			by, query = "isbn_13", isbn
		case 10:
			by, query = "isbn_10", isbn
		}
	}

	var limiter *metadata.RateLimiter
	switch providerKey {
	case "googlebooks":
		limiter = metadata.GoogleBooksLimiter
	case "openlibrary":
		limiter = metadata.OpenLibraryLimiter
	default:
		limiter = metadata.GoogleBooksLimiter
	}

	limiter.Wait()
	result, err := metadata.Search(providers, providerKey, by, query, "")
	if err != nil {
		log.Printf("[Metadata] %s search failed: %v", providerKey, err)
		return nil
	}
	return result
}
