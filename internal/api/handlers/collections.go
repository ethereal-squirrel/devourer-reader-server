package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db/queries"
)

// ListCollections handles GET /library/:id/collections
func (h *Handlers) ListCollections(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	cols, err := queries.ListCollectionsByLibraryPublicOrUser(h.DB, libraryID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "collections": cols})
}

// CreateCollection handles POST /library/:id/collections
func (h *Handlers) CreateCollection(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Title is required"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	col, err := queries.CreateCollection(h.DB, libraryID, uid, body.Title, json.RawMessage(`[]`))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": true, "collection": col})
}

// GetCollection handles GET /library/:id/collections/:collectionId
func (h *Handlers) GetCollection(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("collectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid collection ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	col, err := queries.GetCollectionByID(h.DB, collectionID, uid)
	if err != nil || col.LibraryID != libraryID {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Collection not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "collection": col})
}

// DeleteCollection handles DELETE /collections/:collectionId
func (h *Handlers) DeleteCollection(c *gin.Context) {
	collectionID, err := strconv.ParseInt(c.Param("collectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid collection ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	if err := queries.DeleteCollection(h.DB, collectionID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// AddToCollection handles PATCH /collections/:collectionId/:fileId
func (h *Handlers) AddToCollection(c *gin.Context) {
	collectionID, err := strconv.ParseInt(c.Param("collectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid collection ID"})
		return
	}
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	col, err := queries.GetCollectionByID(h.DB, collectionID, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Collection not found"})
		return
	}

	var fileIDs []int64
	json.Unmarshal(col.Series, &fileIDs)
	for _, id := range fileIDs {
		if id == fileID {
			c.JSON(http.StatusOK, gin.H{"status": true})
			return
		}
	}
	fileIDs = append(fileIDs, fileID)
	newSeries, _ := json.Marshal(fileIDs)
	if err := queries.UpdateCollectionSeries(h.DB, collectionID, json.RawMessage(newSeries)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

// RemoveFromCollection handles DELETE /collections/:collectionId/:fileId
func (h *Handlers) RemoveFromCollection(c *gin.Context) {
	collectionID, err := strconv.ParseInt(c.Param("collectionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid collection ID"})
		return
	}
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid file ID"})
		return
	}
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	col, err := queries.GetCollectionByID(h.DB, collectionID, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Collection not found"})
		return
	}

	var fileIDs []int64
	json.Unmarshal(col.Series, &fileIDs)
	filtered := make([]int64, 0, len(fileIDs))
	for _, id := range fileIDs {
		if id != fileID {
			filtered = append(filtered, id)
		}
	}
	newSeries, _ := json.Marshal(filtered)
	if err := queries.UpdateCollectionSeries(h.DB, collectionID, json.RawMessage(newSeries)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}
