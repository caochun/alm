package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// WorkspaceHandler 工作空间处理器
type WorkspaceHandler struct {
	workspaceRoot string
}

// NewWorkspaceHandler 创建工作空间处理器
func NewWorkspaceHandler(workspaceRoot string) *WorkspaceHandler {
	return &WorkspaceHandler{
		workspaceRoot: workspaceRoot,
	}
}

// ListApplications 列出所有应用
func (h *WorkspaceHandler) ListApplications(c *gin.Context) {
	applications := make([]gin.H, 0)

	// 扫描workspace目录下的所有子目录
	entries, err := os.ReadDir(h.workspaceRoot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appPath := filepath.Join(h.workspaceRoot, entry.Name())
		assetYaml := filepath.Join(appPath, "asset.yaml")

		// 检查是否存在asset.yaml
		if _, err := os.Stat(assetYaml); os.IsNotExist(err) {
			continue
		}

		// 读取asset.yaml获取基本信息
		config, err := h.loadAssetConfig(assetYaml)
		if err != nil {
			continue
		}

		applications = append(applications, gin.H{
			"id":          config["id"],
			"name":        config["name"],
			"description": config["description"],
			"path":        entry.Name(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
	})
}

// loadAssetConfig 加载asset.yaml配置（简化版，只读取基本信息）
func (h *WorkspaceHandler) loadAssetConfig(assetYamlPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(assetYamlPath)
	if err != nil {
		return nil, err
	}

	// 简单解析YAML的前几行获取基本信息
	config := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id:") {
			config["id"] = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		} else if strings.HasPrefix(line, "name:") {
			config["name"] = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			config["description"] = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}

	return config, nil
}

