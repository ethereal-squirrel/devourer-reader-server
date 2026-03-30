package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": "2.0.0"})
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": true})
}

func (h *Handlers) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": true})
}
