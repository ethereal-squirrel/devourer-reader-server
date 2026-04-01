package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/audit"
	"github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	"github.com/devourer/server/internal/metadata"
	"github.com/devourer/server/internal/scanner"
)

// ListLibraries handles GET /libraries
func (h *Handlers) ListLibraries(c *gin.Context) {
	libs, err := queries.ListLibraries(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	type seriesPreview struct {
		ID    int64  `json:"id"`
		Title string `json:"title,omitempty"`
		Cover string `json:"cover,omitempty"`
	}
	type libOut struct {
		ID          int64           `json:"id"`
		Name        string          `json:"name"`
		Path        string          `json:"path"`
		Type        string          `json:"type"`
		Metadata    json.RawMessage `json:"metadata"`
		Series      []seriesPreview `json:"series"`
		SeriesCount int             `json:"seriesCount"`
	}

	out := make([]libOut, 0, len(libs))
	for _, lib := range libs {
		var preview []seriesPreview
		var count int
		switch lib.Type {
		case "book":
			files, _ := queries.ListBookFilesPreview(h.DB, lib.ID, 3)
			count, _ = queries.CountBookFiles(h.DB, lib.ID)
			for _, f := range files {
				preview = append(preview, seriesPreview{ID: f.ID})
			}
		case "audiobook":
			abSeries, _ := queries.ListAudiobookSeriesPreview(h.DB, lib.ID, 3)
			count, _ = queries.CountAudiobookSeries(h.DB, lib.ID)
			for _, s := range abSeries {
				preview = append(preview, seriesPreview{ID: s.ID, Title: s.Title, Cover: s.Cover})
			}
		default:
			series, _ := queries.ListMangaSeriesPreview(h.DB, lib.ID, 3)
			count, _ = queries.CountMangaSeries(h.DB, lib.ID)
			for _, s := range series {
				preview = append(preview, seriesPreview{ID: s.ID, Title: s.Title, Cover: s.Cover})
			}
		}
		if preview == nil {
			preview = []seriesPreview{}
		}
		out = append(out, libOut{
			ID: lib.ID, Name: lib.Name, Path: lib.Path, Type: lib.Type,
			Metadata: lib.Metadata, Series: preview, SeriesCount: count,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "libraries": out})
}

// CreateLibrary handles POST /libraries
func (h *Handlers) CreateLibrary(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Type     string `json:"type"`
		Metadata any    `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" || body.Path == "" || body.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Name, path and type are required"})
		return
	}
	if _, err := os.Stat(body.Path); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Library path does not exist"})
		return
	}

	metaJSON, _ := json.Marshal(body.Metadata)
	lib, err := queries.CreateLibrary(h.DB, body.Name, body.Path, body.Type, json.RawMessage(metaJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}

	audit.LibraryCreated(c.ClientIP(), lib.Name, lib.Path)

	if h.Watcher != nil {
		go h.Watcher.Restart()
	}

	providers, _ := metadata.LoadProviders(h.Cfg.PluginsFS)
	scanCfg := &scanner.Config{
		DB:          h.DB,
		AssetsPath:  h.Cfg.AssetsPath,
		PluginsPath: h.Cfg.PluginsPath,
		Providers:   providers,
	}
	scanner.ScanLibrary(scanCfg, lib.ID)

	c.JSON(http.StatusCreated, gin.H{"status": true, "library": lib})
}

// GetLibrary handles GET /library/:id
func (h *Handlers) GetLibrary(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	lib, err := queries.GetLibraryByID(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)

	var series any
	switch lib.Type {
	case "book":
		files, _ := queries.ListBookFilesByLibrary(h.DB, lib.ID)
		if files == nil {
			files = []*db.BookFile{}
		}
		series = files
	case "audiobook":
		abSeries, _ := queries.ListAudiobookSeriesByLibrary(h.DB, lib.ID)
		if abSeries == nil {
			abSeries = []*db.AudiobookSeries{}
		}
		series = abSeries
	default:
		mSeries, _ := queries.ListMangaSeriesByLibrary(h.DB, lib.ID)
		if mSeries == nil {
			mSeries = []*db.MangaSeries{}
		}
		series = mSeries
	}

	collections, _ := queries.ListCollectionsByLibraryPublicOrUser(h.DB, lib.ID, uid)
	if collections == nil {
		collections = []*db.Collection{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"library": gin.H{
			"id":          lib.ID,
			"name":        lib.Name,
			"path":        lib.Path,
			"type":        lib.Type,
			"metadata":    lib.Metadata,
			"series":      series,
			"collections": collections,
		},
	})
}

// UpdateLibrary handles PATCH /library/:id
func (h *Handlers) UpdateLibrary(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	var body struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Metadata any    `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid request body"})
		return
	}
	metaJSON, _ := json.Marshal(body.Metadata)
	if err := queries.UpdateLibrary(h.DB, id, body.Name, body.Path, json.RawMessage(metaJSON)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	lib, _ := queries.GetLibraryByID(h.DB, id)
	c.JSON(http.StatusOK, gin.H{"status": true, "library": lib})
}

// DeleteLibrary handles DELETE /library/:id
func (h *Handlers) DeleteLibrary(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}

	lib, err := queries.GetLibraryByID(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "Library not found"})
		return
	}

	queries.DeleteRecentlyReadByLibrary(h.DB, id, nil)

	switch lib.Type {
	case "book":
		bookIDs, _ := queries.ListBookFileIDsByLibrary(h.DB, id)
		if len(bookIDs) > 0 {
			queries.DeleteReadingStatusByFileType(h.DB, "book", bookIDs)
			queries.DeleteUserRatingsByFileType(h.DB, "book", bookIDs)
			queries.DeleteUserTagsByFileType(h.DB, "book", bookIDs)
		}
		queries.DeleteBookFilesByLibrary(h.DB, id)
	case "audiobook":
		seriesIDs, _ := queries.ListAudiobookSeriesIDsByLibrary(h.DB, id)
		var abFileIDs []int64
		for _, sid := range seriesIDs {
			files, _ := queries.ListAudiobookFilesPathBySeries(h.DB, sid)
			for _, f := range files {
				abFileIDs = append(abFileIDs, f.ID)
			}
		}
		if len(abFileIDs) > 0 {
			queries.DeleteReadingStatusByFileType(h.DB, "audiobook", abFileIDs)
			queries.DeleteUserRatingsByFileType(h.DB, "audiobook", abFileIDs)
			queries.DeleteUserTagsByFileType(h.DB, "audiobook", abFileIDs)
		}
		if len(seriesIDs) > 0 {
			queries.DeleteAudiobookFilesBySeriesIDs(h.DB, seriesIDs)
		}
		queries.DeleteAudiobookSeriesByLibrary(h.DB, id)
	default:
		seriesIDs, _ := queries.ListMangaSeriesIDsByLibrary(h.DB, id)
		var mangaFileIDs []int64
		for _, sid := range seriesIDs {
			files, _ := queries.ListMangaFilesPathBySeries(h.DB, sid)
			for _, f := range files {
				mangaFileIDs = append(mangaFileIDs, f.ID)
			}
		}
		if len(mangaFileIDs) > 0 {
			queries.DeleteReadingStatusByFileType(h.DB, "manga", mangaFileIDs)
			queries.DeleteUserRatingsByFileType(h.DB, "manga", mangaFileIDs)
			queries.DeleteUserTagsByFileType(h.DB, "manga", mangaFileIDs)
		}
		if len(seriesIDs) > 0 {
			queries.DeleteMangaFilesBySeriesIDs(h.DB, seriesIDs)
		}
		queries.DeleteMangaSeriesByLibrary(h.DB, id)
	}

	queries.DeleteCollectionsByLibrary(h.DB, id)
	queries.DeleteLibrary(h.DB, id)

	os.RemoveAll(lib.MetaBase())

	audit.LibraryDeleted(c.ClientIP(), id)

	if h.Watcher != nil {
		go h.Watcher.Restart()
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}

// RecentlyRead handles GET /recently-read
func (h *Handlers) RecentlyRead(c *gin.Context) {
	userID, _ := c.Get(auth.CtxUserID)
	uid, _ := userID.(int64)
	rr, err := queries.ListRecentlyRead(h.DB, uid, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "recentlyRead": rr})
}

// ScanLibrary handles POST /library/:id/scan
func (h *Handlers) ScanLibrary(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	providers, _ := metadata.LoadProviders(h.Cfg.PluginsFS)
	cfg := &scanner.Config{
		DB:          h.DB,
		AssetsPath:  h.Cfg.AssetsPath,
		PluginsPath: h.Cfg.PluginsPath,
		Providers:   providers,
	}
	result, err := scanner.ScanLibrary(cfg, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": false, "message": err.Error()})
		return
	}
	audit.LibraryScanned(c.ClientIP(), id)
	c.JSON(http.StatusOK, result)
}

// GetScanStatus handles GET /library/:id/scan
func (h *Handlers) GetScanStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "Invalid library ID"})
		return
	}
	c.JSON(http.StatusOK, scanner.GetScanStatus(id))
}
