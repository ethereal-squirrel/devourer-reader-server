package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/db/queries"
)

// CoverImage handles GET /cover-image/:libraryId/:entityId
func (h *Handlers) CoverImage(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	entityParam := strings.TrimSuffix(c.Param("entityId"), ".jpg")
	entityID, err := strconv.ParseInt(entityParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid entity ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	var coverPath string
	if lib.Type == "book" {
		coverPath = filepath.Join(lib.MetaBase(), "files", strconv.FormatInt(entityID, 10), "cover.jpg")
	} else {
		coverPath = filepath.Join(lib.MetaBase(), "series", strconv.FormatInt(entityID, 10), "cover.jpg")
	}

	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Cover not found"})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(coverPath)
}

// PreviewImage handles GET /preview-image/:libraryId/:seriesId/:entityId
func (h *Handlers) PreviewImage(c *gin.Context) {
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
	entityParam := strings.TrimSuffix(c.Param("entityId"), ".jpg")
	entityID, err := strconv.ParseInt(entityParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid entity ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	file, err := queries.GetMangaFileByID(h.DB, entityID)
	if err != nil || file.SeriesID != seriesID {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "File not found"})
		return
	}

	previewPath := filepath.Join(
		lib.MetaBase(), "series",
		strconv.FormatInt(seriesID, 10),
		"previews",
		file.FileName+".jpg",
	)
	if _, err := os.Stat(previewPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Preview not found"})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(previewPath)
}
