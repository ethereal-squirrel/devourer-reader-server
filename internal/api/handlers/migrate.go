package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/migrate"
)

// MigrateCalibre handles POST /migrate/calibre
func (h *Handlers) MigrateCalibre(c *gin.Context) {
	var body struct {
		CalibrePath             string `json:"calibrePath"`
		LibraryName             string `json:"libraryName"`
		LibraryMetadataProvider string `json:"libraryMetadataProvider"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid request body"})
		return
	}
	if body.CalibrePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Calibre path is required"})
		return
	}
	if body.LibraryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Library name is required"})
		return
	}
	if body.LibraryMetadataProvider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Library metadata provider is required"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	d := h.DB
	go func() {
		if err := migrate.MigrateCalibre(d, body.CalibrePath, body.LibraryName, body.LibraryMetadataProvider, uid); err != nil {
			log.Printf("[Migrate] Calibre migration failed: %v", err)
		} else {
			log.Printf("[Migrate] Calibre migration completed for library %q", body.LibraryName)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": true, "message": "Calibre migration started"})
}
