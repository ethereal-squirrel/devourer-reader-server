package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/opds"
)

func baseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// OpdsCatalog handles GET /opds/v1.2/catalog
func (h *Handlers) OpdsCatalog(c *gin.Context) {
	data, err := opds.CatalogFeed(h.DB, baseURL(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/atom+xml;charset=utf-8", data)
}

// OpdsLibraries handles GET /opds/v1.2/libraries
func (h *Handlers) OpdsLibraries(c *gin.Context) {
	h.OpdsCatalog(c)
}

// OpdsLibrary handles GET /opds/v1.2/library/:libraryId
func (h *Handlers) OpdsLibrary(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	data, err := opds.LibraryFeed(h.DB, baseURL(c), libraryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/atom+xml;charset=utf-8", data)
}

// OpdsSearch handles GET /opds/v1.2/library/:libraryId/search
func (h *Handlers) OpdsSearch(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	query := c.Query("q")
	data, err := opds.SearchFeed(h.DB, baseURL(c), libraryID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/atom+xml;charset=utf-8", data)
}

// OpdsBook handles GET /opds/v1.2/library/:libraryId/book/:bookId
func (h *Handlers) OpdsBook(c *gin.Context) {
	libraryID, err := strconv.ParseInt(c.Param("libraryId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	bookID, err := strconv.ParseInt(c.Param("bookId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid book ID"})
		return
	}
	data, err := opds.SingleBookFeed(h.DB, baseURL(c), libraryID, bookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/atom+xml;charset=utf-8", data)
}
