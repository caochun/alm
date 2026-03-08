package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alm/domain"
	"github.com/alm/dsl"
	"github.com/alm/engine"
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

	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app query param is required"})
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

	// Parse deployment env (optional — may not exist yet)
	var deployEnv *domain.DeploymentEnv
	if env != "" {
		deployPath := filepath.Join(h.workspaceRoot, app, "deploy", env+".yaml")
		deployEnv, err = dsl.ParseDeploymentEnv(deployPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse deploy env: " + err.Error()})
			return
		}
	}

	graph := BuildGraph(arch, deployEnv, pipelines)
	c.JSON(http.StatusOK, graph)
}

// GetPlan generates an ExecutionPlan without executing it.
// Query params: app, env, force (optional)
func (h *Handler) GetPlan(c *gin.Context) {
	app := c.Query("app")
	env := c.Query("env")
	if app == "" || env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app and env query params are required"})
		return
	}
	force := c.Query("force") == "true"

	eng := engine.NewEngine(h.workspaceRoot, h.pipelinesDir)
	plan, err := eng.Plan(app, env, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// PostApply generates a plan and executes it (or dry-runs if dryRun=true).
// Query params: app, env, dryRun (optional), force (optional)
func (h *Handler) PostApply(c *gin.Context) {
	app := c.Query("app")
	env := c.Query("env")
	if app == "" || env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app and env query params are required"})
		return
	}
	dryRun := c.Query("dryRun") == "true"
	force := c.Query("force") == "true"

	eng := engine.NewEngine(h.workspaceRoot, h.pipelinesDir)
	plan, err := eng.Apply(context.Background(), app, env, dryRun, force)
	if err != nil {
		// Return the plan even on error so the caller can see which steps failed.
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "plan": plan})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// GetState returns the ReportedState for the given app and environment.
// Query params: app, env
func (h *Handler) GetState(c *gin.Context) {
	app := c.Query("app")
	env := c.Query("env")
	if app == "" || env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app and env query params are required"})
		return
	}

	eng := engine.NewEngine(h.workspaceRoot, h.pipelinesDir)
	state, err := eng.GetState(app, env)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, state)
}

// ── Editor API ───────────────────────────────────────────────────────────────

var validAppName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)

// PipelineInfo is the JSON response for listing available pipeline templates.
type PipelineInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Deliverables []string `json:"deliverables"`
}

// ListPipelines returns available pipeline templates with their deliverables.
func (h *Handler) ListPipelines(c *gin.Context) {
	pipelines, err := dsl.LoadPipelinesFromDir(h.pipelinesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]PipelineInfo, 0, len(pipelines))
	for _, p := range pipelines {
		result = append(result, PipelineInfo{
			Name:         p.Name,
			Description:  p.Description,
			Deliverables: p.Deliverables,
		})
	}
	c.JSON(http.StatusOK, result)
}

// GetArch returns the app architecture as JSON.
func (h *Handler) GetArch(c *gin.Context) {
	app := c.Param("app")
	archPath := filepath.Join(h.workspaceRoot, app, "app-arch.yaml")
	arch, err := dsl.ParseAppArchitecture(archPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, arch)
}

// GetDeployEnv returns the deployment environment as JSON.
func (h *Handler) GetDeployEnv(c *gin.Context) {
	app := c.Param("app")
	env := c.Param("env")
	deployPath := filepath.Join(h.workspaceRoot, app, "deploy", env+".yaml")
	deployEnv, err := dsl.ParseDeploymentEnv(deployPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, deployEnv)
}

// CreateAppRequest is the JSON body for creating a new application.
type CreateAppRequest struct {
	Name string          `json:"name"`
	Arch CreateArchBody  `json:"arch"`
	Env  *CreateEnvBody  `json:"env"`
}

type CreateArchBody struct {
	Description string              `json:"description"`
	Services    []CreateServiceBody `json:"services"`
}

type CreateServiceBody struct {
	Name       string   `json:"name"`
	Pipeline   string   `json:"pipeline"`
	Repository string   `json:"repository"`
	DependsOn  []string `json:"depends_on"`
}

type CreateEnvBody struct {
	EnvName      string                   `json:"envName"`
	Environment  string                   `json:"environment"`
	Services     []CreateServiceDeployBody `json:"services"`
	Dependencies []CreateInfraBody        `json:"dependencies"`
	Bindings     []CreateBindingBody      `json:"bindings"`
	Network      *CreateNetworkBody       `json:"network"`
}

type CreateServiceDeployBody struct {
	Name    string            `json:"name"`
	Accepts string            `json:"accepts"`
	Compute *CreateComputeBody `json:"compute"`
}

type CreateComputeBody struct {
	Type      string              `json:"type"`
	Resources *CreateResourceBody `json:"resources"`
	Ports     []int               `json:"ports"`
}

type CreateResourceBody struct {
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
	Storage  string `json:"storage"`
	Replicas int    `json:"replicas"`
}

type CreateInfraBody struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Provision *CreateProvisionBody   `json:"provision"`
	Resources *CreateResourceBody    `json:"resources"`
	Config    map[string]interface{} `json:"config"`
}

type CreateProvisionBody struct {
	Via      string            `json:"via"`
	Image    string            `json:"image"`
	Env      map[string]string `json:"env"`
	Module   string            `json:"module"`
	Chart    string            `json:"chart"`
	Endpoint string            `json:"endpoint"`
}

type CreateBindingBody struct {
	Service string            `json:"service"`
	Env     map[string]string `json:"env"`
}

type CreateNetworkBody struct {
	Ingress []CreateIngressBody `json:"ingress"`
}

type CreateIngressBody struct {
	Name      string              `json:"name"`
	Type      string              `json:"type"`
	Bind      *CreateBindBody     `json:"bind"`
	TLS       *CreateTLSBody      `json:"tls"`
	Routes    []CreateRouteBody   `json:"routes"`
	Resources *CreateResourceBody `json:"resources"`
}

type CreateBindBody struct {
	IP    string `json:"ip"`
	HTTP  int    `json:"http"`
	HTTPS int    `json:"https"`
}

type CreateTLSBody struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type CreateRouteBody struct {
	Path    string `json:"path"`
	Service string `json:"service"`
	Port    int    `json:"port"`
}

// CreateApp creates a new application with app-arch.yaml and deploy env.
func (h *Handler) CreateApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validAppName.MatchString(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid app name: must start with a letter and contain only letters, digits, and hyphens"})
		return
	}

	appDir := filepath.Join(h.workspaceRoot, req.Name)
	if _, err := os.Stat(filepath.Join(appDir, "app-arch.yaml")); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("app %q already exists", req.Name)})
		return
	}

	arch := buildArchFromRequest(req.Name, &req.Arch)

	archPath := filepath.Join(appDir, "app-arch.yaml")
	if err := dsl.WriteAppArchitecture(archPath, arch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	envName := ""
	if req.Env != nil && req.Env.EnvName != "" {
		deployEnv := buildEnvFromRequest(req.Name, req.Env)
		deployPath := filepath.Join(appDir, "deploy", req.Env.EnvName+".yaml")
		if err := dsl.WriteDeploymentEnv(deployPath, deployEnv); err != nil {
			os.RemoveAll(appDir)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		envName = req.Env.EnvName
	}

	c.JSON(http.StatusCreated, gin.H{"app": req.Name, "env": envName})
}

// SaveArch updates an existing app architecture.
func (h *Handler) SaveArch(c *gin.Context) {
	app := c.Param("app")
	var body CreateArchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	arch := buildArchFromRequest(app, &body)
	archPath := filepath.Join(h.workspaceRoot, app, "app-arch.yaml")
	if err := dsl.WriteAppArchitecture(archPath, arch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// SaveDeployEnv updates an existing deployment environment.
func (h *Handler) SaveDeployEnv(c *gin.Context) {
	app := c.Param("app")
	env := c.Param("env")
	var body CreateEnvBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body.EnvName = env
	deployEnv := buildEnvFromRequest(app, &body)
	deployPath := filepath.Join(h.workspaceRoot, app, "deploy", env+".yaml")
	if err := dsl.WriteDeploymentEnv(deployPath, deployEnv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// CreateEnv creates a new deployment environment for an existing app.
func (h *Handler) CreateEnv(c *gin.Context) {
	app := c.Param("app")
	var body CreateEnvBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify app exists
	archPath := filepath.Join(h.workspaceRoot, app, "app-arch.yaml")
	if _, err := os.Stat(archPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("app %q not found", app)})
		return
	}

	if body.EnvName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "envName is required"})
		return
	}

	deployPath := filepath.Join(h.workspaceRoot, app, "deploy", body.EnvName+".yaml")
	if _, err := os.Stat(deployPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("env %q already exists", body.EnvName)})
		return
	}

	deployEnv := buildEnvFromRequest(app, &body)
	if err := dsl.WriteDeploymentEnv(deployPath, deployEnv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"app": app, "env": body.EnvName})
}

// ── Builder helpers ──────────────────────────────────────────────────────────

func buildArchFromRequest(appName string, body *CreateArchBody) *domain.AppArchitecture {
	arch := &domain.AppArchitecture{
		Name:        appName,
		Description: body.Description,
	}
	for _, s := range body.Services {
		arch.Services = append(arch.Services, &domain.ServiceSpec{
			Name: s.Name, Pipeline: s.Pipeline, Repository: s.Repository, DependsOn: s.DependsOn,
		})
	}
	return arch
}

func buildEnvFromRequest(appName string, body *CreateEnvBody) *domain.DeploymentEnv {
	env := &domain.DeploymentEnv{
		Name:        appName + "-" + body.EnvName,
		Environment: body.Environment,
		App:         appName,
	}

	for _, s := range body.Services {
		sd := &domain.ServiceDeploySpec{Name: s.Name, Accepts: s.Accepts}
		if s.Compute != nil {
			sd.Compute = &domain.ComputeSpec{Type: s.Compute.Type, Ports: s.Compute.Ports}
			if s.Compute.Resources != nil {
				sd.Compute.Resources = &domain.ResourceSpec{
					CPU: s.Compute.Resources.CPU, Memory: s.Compute.Resources.Memory,
					Storage: s.Compute.Resources.Storage, Replicas: s.Compute.Resources.Replicas,
				}
			}
		}
		env.Services = append(env.Services, sd)
	}

	for _, d := range body.Dependencies {
		ir := &domain.InfraResource{Name: d.Name, Type: d.Type, Config: d.Config}
		if d.Resources != nil {
			ir.Resources = &domain.ResourceSpec{
				CPU: d.Resources.CPU, Memory: d.Resources.Memory,
				Storage: d.Resources.Storage, Replicas: d.Resources.Replicas,
			}
		}
		if d.Provision != nil {
			ir.Provision = &domain.InfraProvision{
				Via:      domain.ProvisionVia(d.Provision.Via),
				Image:    d.Provision.Image,
				Env:      d.Provision.Env,
				Module:   d.Provision.Module,
				Chart:    d.Provision.Chart,
				Endpoint: d.Provision.Endpoint,
			}
		}
		env.Dependencies = append(env.Dependencies, ir)
	}

	for _, b := range body.Bindings {
		env.Bindings = append(env.Bindings, &domain.Binding{Service: b.Service, Env: b.Env})
	}

	if body.Network != nil && len(body.Network.Ingress) > 0 {
		env.Network = &domain.NetworkConfig{}
		for _, ig := range body.Network.Ingress {
			spec := &domain.IngressSpec{Name: ig.Name, Type: ig.Type}
			if ig.Bind != nil {
				spec.Bind = &domain.BindSpec{IP: ig.Bind.IP, HTTP: ig.Bind.HTTP, HTTPS: ig.Bind.HTTPS}
			}
			if ig.TLS != nil {
				spec.TLS = &domain.TLSSpec{Cert: ig.TLS.Cert, Key: ig.TLS.Key}
			}
			if ig.Resources != nil {
				spec.Resources = &domain.ResourceSpec{CPU: ig.Resources.CPU, Memory: ig.Resources.Memory}
			}
			for _, r := range ig.Routes {
				spec.Routes = append(spec.Routes, &domain.RouteSpec{Path: r.Path, Service: r.Service, Port: r.Port})
			}
			env.Network.Ingress = append(env.Network.Ingress, spec)
		}
	}

	return env
}
