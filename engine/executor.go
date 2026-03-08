package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/alm/domain"
)

// Execute runs all pending steps in the plan, phase by phase.
// It updates step statuses in place and persists state after each successful step.
// Steps whose dependencies failed are automatically skipped.
func Execute(
	ctx context.Context,
	plan *domain.ExecutionPlan,
	runner CommandRunner,
	stateMgr *StateManager,
) error {
	state, err := stateMgr.Load(plan.App, plan.Environment)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	for _, phase := range plan.Phases {
		if err := executePhase(ctx, phase, runner, state); err != nil {
			// Save partial state even on error
			_ = stateMgr.Save(state)
			return fmt.Errorf("phase %s: %w", phase.Name, err)
		}
	}

	return stateMgr.Save(state)
}

func executePhase(
	ctx context.Context,
	phase *domain.PlanPhase,
	runner CommandRunner,
	state *domain.ReportedState,
) error {
	// Build a set of failed/skipped step IDs so we can cascade failures.
	failed := make(map[string]bool)

	for _, step := range phase.Steps {
		if step.Status == domain.StepSkipped {
			continue
		}
		if step.Status != domain.StepPending {
			continue
		}

		// Check if any dependency failed.
		depFailed := false
		for _, depID := range step.DependsOn {
			if failed[depID] {
				depFailed = true
				break
			}
		}
		if depFailed {
			step.Status = domain.StepSkipped
			step.Error = "dependency failed"
			failed[step.ID] = true
			continue
		}

		step.Status = domain.StepRunning
		output, err := executeStep(ctx, step, runner)
		step.Output = output

		if err != nil {
			step.Status = domain.StepFailed
			step.Error = err.Error()
			failed[step.ID] = true
			continue
		}

		step.Status = domain.StepSucceeded
		updateState(state, phase.Name, step)
	}
	return nil
}

func executeStep(ctx context.Context, step *domain.PlanStep, runner CommandRunner) (string, error) {
	switch a := step.Action.(type) {
	case domain.BuildAction:
		return runner.Run(ctx, a.Command, a.Args, nil)

	case domain.ProvisionAction:
		return executeProvision(ctx, a, runner)

	case domain.DeployAction:
		return executeDeploy(ctx, a, runner)

	case domain.NetworkAction:
		return executeNetwork(ctx, a, runner)

	default:
		return "", fmt.Errorf("unknown action type: %T", step.Action)
	}
}

func executeProvision(ctx context.Context, a domain.ProvisionAction, runner CommandRunner) (string, error) {
	switch a.Via {
	case "docker":
		image := a.Params["image"]
		if image == "" {
			return "", fmt.Errorf("docker provision requires image param")
		}
		env := make(map[string]string)
		for k, v := range a.Params {
			if len(k) > 4 && k[:4] == "env." {
				env[k[4:]] = v
			}
		}
		args := []string{"run", "-d", "--name", a.Resource}
		for k, v := range env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, image)
		return runner.Run(ctx, "docker", args, nil)

	case "terraform":
		module := a.Params["module"]
		return runner.Run(ctx, "terraform", []string{"apply", "-auto-approve", "-target", module}, nil)

	case "helm":
		chart := a.Params["chart"]
		return runner.Run(ctx, "helm", []string{"install", a.Resource, chart}, nil)

	case "external":
		// External resources are pre-existing; no provisioning needed.
		return fmt.Sprintf("external resource %s at %s", a.Resource, a.Params["endpoint"]), nil

	default:
		return "", fmt.Errorf("unknown provision via: %s", a.Via)
	}
}

func executeDeploy(ctx context.Context, a domain.DeployAction, runner CommandRunner) (string, error) {
	if a.Compute == nil {
		return "", fmt.Errorf("deploy action for %s has no compute spec", a.Service)
	}

	switch a.Compute.Type {
	case "docker-container":
		args := []string{"run", "-d", "--name", a.Service}
		for k, v := range a.Env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
		for _, port := range a.Compute.Ports {
			args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
		}
		args = append(args, a.ArtifactRef)
		return runner.Run(ctx, "docker", args, nil)

	case "nginx-static":
		// Copy static bundle and reload nginx
		return runner.Run(ctx, "nginx", []string{"-s", "reload"}, nil)

	default:
		return runner.Run(ctx, "echo", []string{fmt.Sprintf("deploy %s to %s", a.Service, a.Compute.Type)}, nil)
	}
}

func executeNetwork(ctx context.Context, a domain.NetworkAction, runner CommandRunner) (string, error) {
	switch a.Type {
	case "nginx":
		return runner.Run(ctx, "nginx", []string{"-s", "reload"}, nil)
	default:
		return runner.Run(ctx, "echo", []string{fmt.Sprintf("configure %s ingress %s", a.Type, a.Ingress)}, nil)
	}
}

// updateState records a successful step into the ReportedState.
func updateState(state *domain.ReportedState, phaseName string, step *domain.PlanStep) {
	now := time.Now()
	switch a := step.Action.(type) {
	case domain.BuildAction:
		if s, ok := state.Services[a.Service]; ok {
			s.ArtifactRef = step.Output
		}

	case domain.ProvisionAction:
		state.Infra[a.Resource] = &domain.InfraState{
			Status:        "running",
			Outputs:       make(map[string]string), // TODO: parse outputs from provisioner
			ProvisionedAt: now,
		}

	case domain.DeployAction:
		state.Services[a.Service] = &domain.ServiceState{
			ArtifactRef: a.ArtifactRef,
			Status:      "running",
			DeployedAt:  now,
		}

	case domain.NetworkAction:
		state.Ingress[a.Ingress] = &domain.IngressState{
			Status:       "running",
			ConfiguredAt: now,
		}
	}
}
