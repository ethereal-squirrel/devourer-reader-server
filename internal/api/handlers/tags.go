package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db/queries"
)

// ListTags handles GET /tag/:libraryId/:entityId
func (h *Handlers) ListTags(c *gin.Context) {
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

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	tags, err := queries.ListUserTags(h.DB, uid, entityID, lib.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "tags": tags})
}

// CreateTag handles POST /tag/:libraryId/:entityId
func (h *Handlers) CreateTag(c *gin.Context) {
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
		Tag string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Tag) == 0 || len(body.Tag) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid tag"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	if err := queries.CreateUserTag(h.DB, uid, entityID, lib.Type, body.Tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// DeleteTag handles DELETE /tag/:libraryId/:entityId/:tag
func (h *Handlers) DeleteTag(c *gin.Context) {
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
	tag := c.Param("tag")
	if len(tag) == 0 || len(tag) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid tag"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	if err := queries.DeleteUserTag(h.DB, uid, entityID, lib.Type, tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}
