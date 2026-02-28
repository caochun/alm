package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures the Gin router.
func SetupRouter(handler *Handler, webRoot string) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/apps", handler.ListApps)
		v1.GET("/apps/:app/envs", handler.ListEnvs)
		v1.GET("/graph", handler.GetGraph)
	}

	// Serve Web UI static files (if webRoot is provided)
	if webRoot != "" {
		distDir := webRoot
		// If webRoot points to the project root (containing dist/), adjust
		if _, err := os.Stat(filepath.Join(webRoot, "dist")); err == nil {
			distDir = filepath.Join(webRoot, "dist")
		}

		r.Static("/assets", filepath.Join(distDir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(distDir, "favicon.ico"))

		// SPA fallback: all non-API routes serve index.html
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(distDir, "index.html"))
		})
	}

	return r
}
