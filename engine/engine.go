package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/alm/domain"
	"github.com/alm/dsl"
)

// Engine is the top-level facade that combines planning and execution.
type Engine struct {
	WorkspaceRoot string
	PipelinesDir  string
}

// NewEngine creates an Engine with the given workspace and pipelines directory.
func NewEngine(workspaceRoot, pipelinesDir string) *Engine {
	return &Engine{
		WorkspaceRoot: workspaceRoot,
		PipelinesDir:  pipelinesDir,
	}
}

// Plan generates an ExecutionPlan for the given app and environment.
// If force is true, incremental detection is skipped and all steps are pending.
func (e *Engine) Plan(app, env string, force bool) (*domain.ExecutionPlan, error) {
	arch, deployEnv, pipelines, err := e.loadDSL(app, env)
	if err != nil {
		return nil, err
	}

	stateMgr := NewStateManager(e.WorkspaceRoot)
	state, err := stateMgr.Load(app, env)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	return GeneratePlan(arch, deployEnv, pipelines, state, force)
}

// Apply generates a plan and then executes it. When dryRun is true,
// commands are logged but not actually executed.
func (e *Engine) Apply(ctx context.Context, app, env string, dryRun, force bool) (*domain.ExecutionPlan, error) {
	plan, err := e.Plan(app, env, force)
	if err != nil {
		return nil, err
	}

	var runner CommandRunner
	if dryRun {
		runner = &DryRunRunner{}
	} else {
		runner = ShellRunner{}
	}

	stateMgr := NewStateManager(e.WorkspaceRoot)
	if err := Execute(ctx, plan, runner, stateMgr); err != nil {
		return plan, fmt.Errorf("execution error: %w", err)
	}

	return plan, nil
}

// GetState returns the current ReportedState for the given app and environment.
func (e *Engine) GetState(app, env string) (*domain.ReportedState, error) {
	stateMgr := NewStateManager(e.WorkspaceRoot)
	return stateMgr.Load(app, env)
}

// loadDSL parses all three DSL layers for the given app and environment.
func (e *Engine) loadDSL(app, env string) (*domain.AppArchitecture, *domain.DeploymentEnv, map[string]*domain.Pipeline, error) {
	archPath := filepath.Join(e.WorkspaceRoot, app, "app-arch.yaml")
	arch, err := dsl.ParseAppArchitecture(archPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse app architecture: %w", err)
	}

	deployPath := filepath.Join(e.WorkspaceRoot, app, "deploy", env+".yaml")
	deployEnv, err := dsl.ParseDeploymentEnv(deployPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse deployment env: %w", err)
	}

	pipelines, err := dsl.LoadPipelinesFromDir(e.PipelinesDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load pipelines: %w", err)
	}

	// Run cross-model validation.
	if errs := dsl.Validate(deployEnv, arch, pipelines); len(errs) > 0 {
		return nil, nil, nil, fmt.Errorf("validation failed: %v", errs[0])
	}

	return arch, deployEnv, pipelines, nil
}
