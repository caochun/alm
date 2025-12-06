package api

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(handler *Handler, workspaceHandler *WorkspaceHandler, fileHandler *FileHandler, workspaceRoot string, webRoot string) *gin.Engine {
	router := gin.Default()

	// CORS中间件
	router.Use(corsMiddleware())

	// API路由组
	api := router.Group("/api/v1")
	{
		// 工作空间相关
		api.GET("/workspace/applications", workspaceHandler.ListApplications)

		// 资产相关
		api.GET("/asset", handler.GetAsset)
		api.GET("/asset/states", handler.GetStates)
		api.GET("/asset/current-state", handler.GetCurrentState)
		api.GET("/asset/assets", handler.GetAssets)
		api.GET("/asset/history", handler.GetTransitionHistory)
		api.GET("/asset/graph", handler.GetStateMachineGraph)

		// 状态转换
		api.POST("/asset/transition", handler.Transition)

		// 文件浏览相关
		api.GET("/files", fileHandler.ListFiles)
		api.GET("/files/content", fileHandler.GetFileContent)
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 静态文件服务（Web UI）
	if webRoot != "" {
		distPath := filepath.Join(webRoot, "dist")

		// 提供assets目录的静态文件
		router.Static("/assets", filepath.Join(distPath, "assets"))

		// 提供index.html
		router.StaticFile("/", filepath.Join(distPath, "index.html"))

		// 处理前端路由（SPA fallback）
		router.NoRoute(func(c *gin.Context) {
			// 如果是API请求，返回404
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
				return
			}
			// 否则返回index.html（让前端路由处理）
			c.File(filepath.Join(distPath, "index.html"))
		})
	}

	return router
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
