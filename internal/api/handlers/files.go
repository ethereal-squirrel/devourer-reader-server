package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db/queries"
	"github.com/devourer/server/internal/scanner"
)

// GetFile handles GET /file/:libraryId/:id
func (h *Handlers) GetFile(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	if lib.Type == "book" {
		file, err := queries.GetBookFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": true, "file": file})
	} else {
		file, err := queries.GetMangaFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}

		type withNext struct {
			ID          int64  `json:"id"`
			SeriesID    int64  `json:"series_id"`
			FileName    string `json:"file_name"`
			FileFormat  string `json:"file_format"`
			Volume      int    `json:"volume"`
			Chapter     int    `json:"chapter"`
			TotalPages  int    `json:"total_pages"`
			CurrentPage int    `json:"current_page"`
			IsRead      bool   `json:"is_read"`
			NextFile    any    `json:"nextFile"`
		}
		var nextFile any
		if file.Volume > 0 {
			nf, err := queries.GetMangaFileBySeriesAndVolume(h.DB, file.SeriesID, file.Volume+1)
			if err == nil {
				nextFile = gin.H{"id": nf.ID, "series_id": nf.SeriesID}
			}
		} else if file.Chapter > 0 {
			nf, err := queries.GetMangaFileBySeriesAndChapter(h.DB, file.SeriesID, file.Chapter+1)
			if err == nil {
				nextFile = gin.H{"id": nf.ID, "series_id": nf.SeriesID}
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": true, "file": withNext{
			ID: file.ID, SeriesID: file.SeriesID, FileName: file.FileName,
			FileFormat: file.FileFormat, Volume: file.Volume, Chapter: file.Chapter,
			TotalPages: file.TotalPages, CurrentPage: file.CurrentPage,
			IsRead: file.IsRead, NextFile: nextFile,
		}})
	}
}

// StreamFile handles GET /stream/:libraryId/:id
func (h *Handlers) StreamFile(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	var filePath, fileName, contentType string

	if lib.Type == "book" {
		file, err := queries.GetBookFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}
		filePath = file.Path
		fileName = file.FileName
		contentType = "application/octet-stream"
	} else {
		file, err := queries.GetMangaFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}
		filePath = file.Path
		fileName = file.FileName
		switch file.FileFormat {
		case "cbz", "zip":
			contentType = "application/zip"
		case "cbr", "rar":
			contentType = "application/x-rar-compressed"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Unsupported file format"})
			return
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found on disk"})
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to stat file"})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+filepath.Base(fileName)+`"`)
	c.Header("Content-Type", contentType)
	c.Header("X-File-Size", strconv.FormatInt(stat.Size(), 10))
	http.ServeContent(c.Writer, c.Request, fileName, stat.ModTime(), f)
}

// ScanFile handles POST /file/:id/scan
func (h *Handlers) ScanFile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}

	book, err := queries.GetBookFileByID(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(book.Path), ".epub") {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "Not an EPUB file"})
		return
	}

	epubMeta, err := scanner.ScanEpub(book.Path)
	if err != nil || epubMeta == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "Could not read EPUB metadata"})
		return
	}

	meta := map[string]any{
		"title":       epubMeta.Title,
		"author":      epubMeta.Author,
		"publisher":   epubMeta.Publisher,
		"date":        epubMeta.Date,
		"description": epubMeta.Description,
		"language":    epubMeta.Language,
		"isbn":        epubMeta.ISBN,
	}
	metaJSON, _ := json.Marshal(meta)
	queries.UpdateBookFileMetadata(h.DB, id, json.RawMessage(metaJSON))
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// MarkAsRead handles POST /file/:libraryId/:id/mark-as-read
func (h *Handlers) MarkAsRead(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	var totalPages int
	if lib.Type == "book" {
		file, err := queries.GetBookFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}
		totalPages = file.TotalPages
	} else {
		file, err := queries.GetMangaFileByID(h.DB, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
			return
		}
		totalPages = file.TotalPages
	}

	if err := queries.UpsertReadingStatus(h.DB, uid, id, lib.Type, strconv.Itoa(totalPages)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// UnmarkAsRead handles DELETE /file/:libraryId/:id/mark-as-read
func (h *Handlers) UnmarkAsRead(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	if err := queries.DeleteReadingStatus(h.DB, uid, id, lib.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// PageEvent handles POST /file/page-event
func (h *Handlers) PageEvent(c *gin.Context) {
	var body struct {
		LibraryID int64 `json:"libraryId"`
		FileID    int64 `json:"fileId"`
		Page      int   `json:"page"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.FileID == 0 || body.LibraryID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "libraryId, fileId and page are required"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	lib, err := queries.GetLibraryByID(h.DB, body.LibraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	pageStr := strconv.Itoa(body.Page)
	if err := queries.UpsertReadingStatus(h.DB, uid, body.FileID, lib.Type, pageStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	go scanner.UpdateRecentlyRead(h.DB, lib, body.FileID, pageStr, uid)
	c.JSON(http.StatusOK, gin.H{"status": true})
}
