package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	workspaceRoot string
}

// NewFileHandler 创建文件处理器
func NewFileHandler(workspaceRoot string) *FileHandler {
	return &FileHandler{
		workspaceRoot: workspaceRoot,
	}
}

// FileInfo 文件信息
type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"` // "file" or "directory"
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// ListFiles 列出目录下的文件
func (h *FileHandler) ListFiles(c *gin.Context) {
	appPath := c.Query("appPath")
	filePath := c.Query("path") // 相对路径，相对于应用目录

	if appPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appPath parameter is required"})
		return
	}

	// 构建完整路径
	basePath := filepath.Join(h.workspaceRoot, appPath)
	
	// 如果指定了path，则相对于应用目录
	if filePath != "" {
		// 安全检查：防止路径遍历攻击
		if strings.Contains(filePath, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
			return
		}
		basePath = filepath.Join(basePath, filePath)
	}

	// 检查路径是否存在
	info, err := os.Stat(basePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
		return
	}

	// 如果是文件，返回文件内容
	if !info.IsDir() {
		content, err := os.ReadFile(basePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"type":    "file",
			"path":    filePath,
			"content": string(content),
			"size":    info.Size(),
		})
		return
	}

	// 如果是目录，列出文件
	entries, err := os.ReadDir(basePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	files := make([]FileInfo, 0)
	for _, entry := range entries {
		// 跳过隐藏文件（可选）
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := FileInfo{
			Name:     entry.Name(),
			Path:     filepath.Join(filePath, entry.Name()),
			Type:     "file",
			Size:     info.Size(),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
		}

		if entry.IsDir() {
			fileInfo.Type = "directory"
			fileInfo.Size = 0
		}

		files = append(files, fileInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"type":  "directory",
		"path":  filePath,
		"files": files,
	})
}

// GetFileContent 获取文件内容
func (h *FileHandler) GetFileContent(c *gin.Context) {
	appPath := c.Query("appPath")
	filePath := c.Query("path")

	if appPath == "" || filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appPath and path parameters are required"})
		return
	}

	// 安全检查
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	fullPath := filepath.Join(h.workspaceRoot, appPath, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    filePath,
		"content": string(content),
	})
}

