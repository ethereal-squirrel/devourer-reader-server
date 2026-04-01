package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/metadata"
)

// ListMetadataProviders handles GET /metadata/providers
func (h *Handlers) ListMetadataProviders(c *gin.Context) {
	providers, err := metadata.LoadProviders(h.Cfg.PluginsFS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	list := make([]*metadata.Provider, 0, len(providers))
	for _, p := range providers {
		list = append(list, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "providers": list})
}

// SearchMetadata handles POST /metadata/search
func (h *Handlers) SearchMetadata(c *gin.Context) {
	var body struct {
		Provider string `json:"provider"`
		By       string `json:"by"`
		Value    string `json:"value"`
		APIKey   string `json:"apiKey"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Provider == "" || body.By == "" || body.Value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "provider, by, and value are required"})
		return
	}

	providers, err := metadata.LoadProviders(h.Cfg.PluginsFS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	result, err := metadata.Search(providers, body.Provider, body.By, body.Value, body.APIKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "result": result})
}

// SearchAudiobookMetadata handles GET /metadata/audiobooks/search?q=<query>&region=<region>
func (h *Handlers) SearchAudiobookMetadata(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "q is required"})
		return
	}
	region := c.Query("region")

	results, err := metadata.AudibleSearchAll(query, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "results": results})
}
