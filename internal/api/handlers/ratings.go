package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db/queries"
)

// RateEntity handles POST /rate/:libraryId/:entityId
func (h *Handlers) RateEntity(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	entityID, err := strconv.ParseInt(c.Param("entityId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid entity ID"})
		return
	}

	var body struct {
		Rating int `json:"rating"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Rating < 0 || body.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid rating"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	if err := queries.UpsertUserRating(h.DB, uid, entityID, lib.Type, body.Rating); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}
