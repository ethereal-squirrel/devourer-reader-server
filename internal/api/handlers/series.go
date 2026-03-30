package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/audit"
	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	imgconvert "github.com/devourer/server/internal/image"
	"github.com/devourer/server/internal/metadata"
	"github.com/devourer/server/internal/scanner"
)

// GetSeries handles GET /series/:libraryId/:seriesId
func (h *Handlers) GetSeries(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	seriesID, err := strconv.ParseInt(c.Param("seriesId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid series ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	if lib.Type == "book" {
		file, err := queries.GetBookFileByID(h.DB, seriesID)
		if err != nil || file.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Book not found"})
			return
		}
		rating, _ := queries.GetUserRating(h.DB, uid, file.ID, lib.Type)
		tags, _ := queries.ListUserTags(h.DB, uid, file.ID, lib.Type)
		c.JSON(http.StatusOK, gin.H{"status": true, "series": gin.H{
			"id": file.ID, "title": file.Title, "path": file.Path,
			"file_name": file.FileName, "file_format": file.FileFormat,
			"total_pages": file.TotalPages, "is_read": file.IsRead,
			"library_id": file.LibraryID, "metadata": file.Metadata,
			"formats": file.Formats, "tags": file.Tags,
			"userRating": rating, "userTags": tags,
		}})
	} else {
		series, err := queries.GetMangaSeriesByID(h.DB, seriesID)
		if err != nil || series.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Series not found"})
			return
		}
		rating, _ := queries.GetUserRating(h.DB, uid, series.ID, lib.Type)
		tags, _ := queries.ListUserTags(h.DB, uid, series.ID, lib.Type)
		c.JSON(http.StatusOK, gin.H{"status": true, "series": gin.H{
			"id": series.ID, "title": series.Title, "path": series.Path,
			"cover": series.Cover, "library_id": series.LibraryID,
			"manga_data": series.MangaData,
			"userRating": rating, "userTags": tags,
		}})
	}
}

// ListSeriesFiles handles GET /series/:libraryId/:seriesId/files
func (h *Handlers) ListSeriesFiles(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	seriesID, err := strconv.ParseInt(c.Param("seriesId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid series ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	files, err := queries.ListMangaFilesBySeries(h.DB, seriesID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	type fileOut struct {
		ID          int64  `json:"id"`
		SeriesID    int64  `json:"series_id"`
		FileName    string `json:"file_name"`
		FileFormat  string `json:"file_format"`
		Volume      int    `json:"volume"`
		Chapter     int    `json:"chapter"`
		TotalPages  int    `json:"total_pages"`
		CurrentPage int    `json:"current_page"`
	}
	out := make([]fileOut, 0, len(files))
	for _, f := range files {
		status, _ := queries.GetReadingStatus(h.DB, uid, f.ID, lib.Type)
		cp := 0
		if status != nil {
			cp, _ = strconv.Atoi(status.CurrentPage)
		}
		out = append(out, fileOut{
			ID: f.ID, SeriesID: f.SeriesID, FileName: f.FileName,
			FileFormat: f.FileFormat, Volume: f.Volume, Chapter: f.Chapter,
			TotalPages: f.TotalPages, CurrentPage: cp,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "files": out})
}

// UpdateSeriesMetadata handles PATCH /series/:libraryId/:seriesId/metadata
func (h *Handlers) UpdateSeriesMetadata(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	seriesID, err := strconv.ParseInt(c.Param("seriesId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid series ID"})
		return
	}

	var body struct {
		Metadata any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Metadata == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Metadata is required"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	metaJSON, _ := json.Marshal(body.Metadata)

	if lib.Type == "book" {
		book, err := queries.GetBookFileByID(h.DB, seriesID)
		if err != nil || book.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Book not found"})
			return
		}
		if err := queries.UpdateBookFileMetadata(h.DB, seriesID, json.RawMessage(metaJSON)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
			return
		}
	} else {
		series, err := queries.GetMangaSeriesByID(h.DB, seriesID)
		if err != nil || series.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Series not found"})
			return
		}
		if err := queries.UpdateMangaSeriesMetadata(h.DB, seriesID, json.RawMessage(metaJSON)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
			return
		}

		var meta map[string]any
		if json.Unmarshal(metaJSON, &meta) == nil {
			if coverURL, ok := meta["coverImage"].(string); ok && coverURL != "" {
				coverDir := filepath.Join(lib.MetaBase(), "series", strconv.FormatInt(seriesID, 10))
				os.MkdirAll(coverDir, 0o755)
				coverPath := filepath.Join(coverDir, "cover.jpg")
				go imgconvert.DownloadAndSave(coverURL, coverPath, imgconvert.CoverMaxWidth)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}

// UpdateSeriesCover handles PATCH /series/:libraryId/:seriesId/cover
func (h *Handlers) UpdateSeriesCover(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	seriesID, err := strconv.ParseInt(c.Param("seriesId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid series ID"})
		return
	}

	uploadedFile, _ := c.FormFile("cover")
	coverURL := c.PostForm("cover")
	if uploadedFile == nil && coverURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Cover image file or URL is required"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	var coverDir string
	if lib.Type == "book" {
		book, err := queries.GetBookFileByID(h.DB, seriesID)
		if err != nil || book.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Book not found"})
			return
		}
		coverDir = filepath.Join(lib.MetaBase(), "files", strconv.FormatInt(seriesID, 10))
	} else {
		series, err := queries.GetMangaSeriesByID(h.DB, seriesID)
		if err != nil || series.LibraryID != libraryID {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Series not found"})
			return
		}
		coverDir = filepath.Join(lib.MetaBase(), "series", strconv.FormatInt(seriesID, 10))
	}

	os.MkdirAll(coverDir, 0o755)
	coverPath := filepath.Join(coverDir, "cover.jpg")

	if uploadedFile != nil {
		f, err := uploadedFile.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to open uploaded file"})
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to read uploaded file"})
			return
		}
		if err := imgconvert.ResizeAndSave(data, coverPath, imgconvert.CoverMaxWidth); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to save cover image"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": true, "message": "Cover uploaded successfully"})
	} else {
		if err := imgconvert.DownloadAndSave(coverURL, coverPath, imgconvert.CoverMaxWidth); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to download cover image"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": true, "message": "Cover downloaded successfully"})
	}
}

// UploadBook handles PUT /book/:libraryId
func (h *Handlers) UploadBook(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}
	if lib.Type != "book" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "This endpoint does not support manga uploads"})
		return
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "File is required"})
		return
	}

	safeFilename := filepath.Base(uploadedFile.Filename)
	ext := strings.ToLower(filepath.Ext(safeFilename))
	if !h.Cfg.UploadAllowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": fmt.Sprintf("File extension %q is not allowed", ext)})
		return
	}
	maxBytes := h.Cfg.UploadMaxSizeMB * 1024 * 1024
	if uploadedFile.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": fmt.Sprintf("File exceeds maximum size of %d MB", h.Cfg.UploadMaxSizeMB)})
		return
	}

	destPath := filepath.Join(lib.Path, safeFilename)
	if !strings.HasPrefix(filepath.Clean(destPath)+string(filepath.Separator), filepath.Clean(lib.Path)+string(filepath.Separator)) {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file path"})
		return
	}
	if err := c.SaveUploadedFile(uploadedFile, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to save file"})
		return
	}

	providers, _ := metadata.LoadProviders(h.Cfg.PluginsPath)
	cfg := &scanner.Config{
		DB:          h.DB,
		AssetsPath:  h.Cfg.AssetsPath,
		PluginsPath: h.Cfg.PluginsPath,
		Providers:   providers,
	}
	entity, err := scanner.ProcessBook(cfg, lib, destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	audit.FileUploaded(c.ClientIP(), libraryID, safeFilename)
	c.JSON(http.StatusOK, gin.H{"status": true, "entity": entity})
}

// CreateMangaSeries handles PUT /series
func (h *Handlers) CreateMangaSeries(c *gin.Context) {
	var body struct {
		Payload struct {
			LibraryID int64           `json:"library_id"`
			Title     string          `json:"title"`
			Path      string          `json:"path"`
			Cover     string          `json:"cover"`
			MangaData json.RawMessage `json:"manga_data"`
		} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Payload.LibraryID == 0 || body.Payload.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Payload with library_id and title is required"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, body.Payload.LibraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}
	if lib.Type != "manga" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "This endpoint does not support book uploads"})
		return
	}

	mangaData := body.Payload.MangaData
	if len(mangaData) == 0 {
		mangaData = json.RawMessage(`{}`)
	}

	ms := &db.MangaSeries{
		Title:     body.Payload.Title,
		Path:      body.Payload.Path,
		Cover:     body.Payload.Cover,
		LibraryID: body.Payload.LibraryID,
		MangaData: mangaData,
	}
	entity, err := queries.CreateMangaSeries(h.DB, ms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "entity": entity})
}

// UploadMangaFile handles PUT /series/:libraryId/:seriesId/file
func (h *Handlers) UploadMangaFile(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	seriesID, err := strconv.ParseInt(c.Param("seriesId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid series ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}
	if lib.Type != "manga" {
		c.JSON(http.StatusOK, gin.H{"status": true, "message": "This endpoint does not support book uploads"})
		return
	}

	series, err := queries.GetMangaSeriesByID(h.DB, seriesID)
	if err != nil || series.LibraryID != libraryID {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Series not found"})
		return
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "File is required"})
		return
	}

	safeFilename := filepath.Base(uploadedFile.Filename)
	ext := strings.ToLower(filepath.Ext(safeFilename))
	if !h.Cfg.UploadAllowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": fmt.Sprintf("File extension %q is not allowed", ext)})
		return
	}
	maxBytes := h.Cfg.UploadMaxSizeMB * 1024 * 1024
	if uploadedFile.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": fmt.Sprintf("File exceeds maximum size of %d MB", h.Cfg.UploadMaxSizeMB)})
		return
	}

	seriesDir := series.Path
	if seriesDir == "" {
		seriesDir = filepath.Join(lib.Path, series.Title)
	}
	os.MkdirAll(seriesDir, 0o755)
	destPath := filepath.Join(seriesDir, safeFilename)
	if !strings.HasPrefix(filepath.Clean(destPath)+string(filepath.Separator), filepath.Clean(lib.Path)+string(filepath.Separator)) {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file path"})
		return
	}
	if err := c.SaveUploadedFile(uploadedFile, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Failed to save file"})
		return
	}

	providers, _ := metadata.LoadProviders(h.Cfg.PluginsPath)
	cfg := &scanner.Config{
		DB:          h.DB,
		AssetsPath:  h.Cfg.AssetsPath,
		PluginsPath: h.Cfg.PluginsPath,
		Providers:   providers,
	}
	if err := scanner.ProcessManga(cfg, lib, series.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	audit.FileUploaded(c.ClientIP(), libraryID, safeFilename)
	c.JSON(http.StatusOK, gin.H{"status": true})
}
