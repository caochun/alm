package api

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures the Gin router.
func SetupRouter(handler *Handler, webRoot string) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
		v1.POST("/apps", handler.CreateApp)
		v1.GET("/apps/:app/envs", handler.ListEnvs)
		v1.POST("/apps/:app/envs", handler.CreateEnv)
		v1.GET("/apps/:app/arch", handler.GetArch)
		v1.PUT("/apps/:app/arch", handler.SaveArch)
		v1.GET("/apps/:app/envs/:env", handler.GetDeployEnv)
		v1.PUT("/apps/:app/envs/:env", handler.SaveDeployEnv)
		v1.GET("/pipelines", handler.ListPipelines)
		v1.GET("/graph", handler.GetGraph)
		v1.POST("/plan", handler.GetPlan)
		v1.POST("/apply", handler.PostApply)
		v1.GET("/state", handler.GetState)
	}

	// Serve static files (plain HTML/JS/CSS, no build step)
	if webRoot != "" {
		r.StaticFile("/", filepath.Join(webRoot, "index.html"))
		r.StaticFile("/app.js", filepath.Join(webRoot, "app.js"))
		r.StaticFile("/editor-shared.js", filepath.Join(webRoot, "editor-shared.js"))
		r.StaticFile("/arch-editor.js", filepath.Join(webRoot, "arch-editor.js"))
		r.StaticFile("/deploy-editor.js", filepath.Join(webRoot, "deploy-editor.js"))
		r.StaticFile("/favicon.ico", filepath.Join(webRoot, "favicon.ico"))

		// Fallback: serve index.html for any non-API route
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(webRoot, "index.html"))
		})
	}

	return r
}
