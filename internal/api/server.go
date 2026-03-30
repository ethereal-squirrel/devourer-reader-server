package api

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/devourer/server/internal/api/handlers"
	authmw "github.com/devourer/server/internal/auth"
	"github.com/devourer/server/internal/config"
)

func NewServer(d *sql.DB, cfg *config.Config, w handlers.Watcher, clientFS fs.FS) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "X-File-Size, Content-Range, Content-Length")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.Static("/assets", cfg.AssetsPath)

	clientHTTPFS := http.FS(clientFS)
	spaHandler := func(c *gin.Context) {
		file := c.Param("filepath")
		f, err := clientHTTPFS.Open(file)
		if err != nil {
			c.FileFromFS("/", clientHTTPFS)
			return
		}
		f.Close()
		c.FileFromFS(file, clientHTTPFS)
	}
	r.GET("/client", func(c *gin.Context) { c.FileFromFS("/", clientHTTPFS) })
	r.GET("/client/*filepath", spaHandler)

	h := handlers.New(d, cfg)
	h.Watcher = w

	auth := authmw.CheckAuth(d)

	// Root
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "Devourer Server") })
	r.GET("/version", h.Version)
	r.GET("/health", h.Health)
	r.GET("/status", auth, h.Status)

	// Auth
	r.POST("/login", authmw.LoginRateLimit(), h.Login)
	r.POST("/users", auth, authmw.RequirePermission("CreateUser"), h.CreateUser)
	r.GET("/roles", auth, h.GetRoles)
	r.GET("/users", auth, h.ListUsers)
	r.DELETE("/user/:id", auth, authmw.RequirePermission("CreateUser"), h.DeleteUser)
	r.PATCH("/user/:id", auth, authmw.RequirePermission("CreateUser"), h.EditUser)

	// Libraries
	r.GET("/libraries", auth, h.ListLibraries)
	r.POST("/libraries", auth, authmw.RequirePermission("ManageLibrary"), h.CreateLibrary)
	r.GET("/recently-read", auth, h.RecentlyRead)
	r.GET("/library/:id", auth, h.GetLibrary)
	r.PATCH("/library/:id", auth, authmw.RequirePermission("ManageLibrary"), h.UpdateLibrary)
	r.DELETE("/library/:id", auth, authmw.RequirePermission("ManageLibrary"), h.DeleteLibrary)
	r.POST("/library/:id/scan", auth, authmw.RequirePermission("AddFile"), h.ScanLibrary)
	r.GET("/library/:id/scan", auth, h.GetScanStatus)

	// Collections
	r.GET("/library/:id/collections", auth, h.ListCollections)
	r.POST("/library/:id/collections", auth, authmw.RequirePermission("ManageCollections"), h.CreateCollection)
	r.GET("/library/:id/collections/:collectionId", auth, h.GetCollection)
	r.DELETE("/collections/:collectionId", auth, authmw.RequirePermission("ManageCollections"), h.DeleteCollection)
	r.PATCH("/collections/:collectionId/:fileId", auth, authmw.RequirePermission("ManageCollections"), h.AddToCollection)
	r.DELETE("/collections/:collectionId/:fileId", auth, authmw.RequirePermission("ManageCollections"), h.RemoveFromCollection)

	// Series
	r.GET("/series/:libraryId/:seriesId", auth, h.GetSeries)
	r.GET("/series/:libraryId/:seriesId/files", auth, h.ListSeriesFiles)
	r.PATCH("/series/:libraryId/:seriesId/metadata", auth, authmw.RequirePermission("EditMetadata"), h.UpdateSeriesMetadata)
	r.PATCH("/series/:libraryId/:seriesId/cover", auth, authmw.RequirePermission("EditMetadata"), h.UpdateSeriesCover)
	r.PUT("/book/:libraryId", auth, authmw.RequirePermission("AddFile"), h.UploadBook)
	r.PUT("/series", auth, authmw.RequirePermission("AddFile"), h.CreateMangaSeries)
	r.PUT("/series/:libraryId/:seriesId/file", auth, authmw.RequirePermission("AddFile"), h.UploadMangaFile)

	// Files
	r.GET("/file/:libraryId/:id", auth, h.GetFile)
	r.GET("/stream/:libraryId/:id", h.StreamFile)
	r.POST("/file/:libraryId/:id/scan", auth, h.ScanFile)
	r.POST("/file/:libraryId/:id/mark-as-read", auth, h.MarkAsRead)
	r.DELETE("/file/:libraryId/:id/mark-as-read", auth, h.UnmarkAsRead)
	r.POST("/file/page-event", auth, h.PageEvent)

	// Images
	r.GET("/cover-image/:libraryId/:entityId", h.CoverImage)
	r.GET("/preview-image/:libraryId/:seriesId/:entityId", h.PreviewImage)

	// Ratings
	r.POST("/rate/:libraryId/:entityId", auth, h.RateEntity)

	// Tags
	r.GET("/tag/:libraryId/:entityId", auth, h.ListTags)
	r.POST("/tag/:libraryId/:entityId", auth, h.CreateTag)
	r.DELETE("/tag/:libraryId/:entityId/:tag", auth, h.DeleteTag)

	// Metadata
	r.GET("/metadata/providers", auth, h.ListMetadataProviders)
	r.POST("/metadata/search", auth, h.SearchMetadata)

	// Calibre migration
	r.POST("/migrate/calibre", auth, authmw.RequirePermission("ManageLibrary"), h.MigrateCalibre)

	// OPDS 1.2
	opds := r.Group("/opds/v1.2")
	{
		opds.GET("/catalog", h.OpdsCatalog)
		opds.GET("/libraries", h.OpdsLibraries)
		opds.GET("/library/:libraryId", h.OpdsLibrary)
		opds.GET("/library/:libraryId/search", h.OpdsSearch)
		opds.GET("/library/:libraryId/book/:bookId", h.OpdsBook)
	}

	return r
}
