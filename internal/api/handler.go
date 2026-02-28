package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alm/dsl"
	"github.com/gin-gonic/gin"
)

// Handler holds the workspace root and pipeline templates directory.
type Handler struct {
	workspaceRoot string
	pipelinesDir  string
}

// NewHandler creates a new Handler.
func NewHandler(workspaceRoot, pipelinesDir string) *Handler {
	return &Handler{
		workspaceRoot: workspaceRoot,
		pipelinesDir:  pipelinesDir,
	}
}

// ListApps scans the workspace directory and returns application names
// (subdirectories that contain an app-arch.yaml file).
func (h *Handler) ListApps(c *gin.Context) {
	entries, err := os.ReadDir(h.workspaceRoot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var apps []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		archPath := filepath.Join(h.workspaceRoot, e.Name(), "app-arch.yaml")
		if _, err := os.Stat(archPath); err == nil {
			apps = append(apps, e.Name())
		}
	}
	if apps == nil {
		apps = []string{}
	}
	c.JSON(http.StatusOK, apps)
}

// ListEnvs returns the available deployment environments for an app
// by scanning workspace/:app/deploy/*.yaml files.
func (h *Handler) ListEnvs(c *gin.Context) {
	app := c.Param("app")
	deployDir := filepath.Join(h.workspaceRoot, app, "deploy")

	entries, err := os.ReadDir(deployDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, []string{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var envs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			envName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
			envs = append(envs, envName)
		}
	}
	if envs == nil {
		envs = []string{}
	}
	c.JSON(http.StatusOK, envs)
}

// GetGraph parses the DSL files and returns graph data for visualization.
// Query params: app, env
func (h *Handler) GetGraph(c *gin.Context) {
	app := c.Query("app")
	env := c.Query("env")

	if app == "" || env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app and env query params are required"})
		return
	}

	// Load pipeline templates
	pipelines, err := dsl.LoadPipelinesFromDir(h.pipelinesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load pipelines: " + err.Error()})
		return
	}

	// Parse app architecture
	archPath := filepath.Join(h.workspaceRoot, app, "app-arch.yaml")
	arch, err := dsl.ParseAppArchitecture(archPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse app-arch.yaml: " + err.Error()})
		return
	}

	// Parse deployment env
	deployPath := filepath.Join(h.workspaceRoot, app, "deploy", env+".yaml")
	deployEnv, err := dsl.ParseDeploymentEnv(deployPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse deploy env: " + err.Error()})
		return
	}

	graph := BuildGraph(arch, deployEnv, pipelines)
	c.JSON(http.StatusOK, graph)
}
