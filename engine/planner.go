package engine

import (
	"fmt"
	"time"

	"github.com/alm/domain"
)

// GeneratePlan creates an ExecutionPlan by comparing DSL desired state against
// the current ReportedState. Steps whose configHash matches are marked skipped.
// When force is true, all steps are generated as pending regardless of state.
func GeneratePlan(
	arch *domain.AppArchitecture,
	env *domain.DeploymentEnv,
	pipelines map[string]*domain.Pipeline,
	state *domain.ReportedState,
	force bool,
) (*domain.ExecutionPlan, error) {

	plan := &domain.ExecutionPlan{
		App:         arch.Name,
		Environment: env.Environment,
		CreatedAt:   time.Now(),
	}

	// Determine topological order for services.
	ordered, err := arch.TopologicalOrder()
	if err != nil {
		return nil, fmt.Errorf("topological sort: %w", err)
	}

	buildPhase, err := generateBuildPhase(ordered, env, pipelines, state, force)
	if err != nil {
		return nil, err
	}

	provisionPhase := generateProvisionPhase(env, state, force)

	deployPhase, err := generateDeployPhase(ordered, env, pipelines, state, force)
	if err != nil {
		return nil, err
	}

	networkPhase := generateNetworkPhase(env, state, force)

	plan.Phases = []*domain.PlanPhase{buildPhase, provisionPhase, deployPhase, networkPhase}
	return plan, nil
}

// generateBuildPhase creates build steps for each service that needs rebuilding.
func generateBuildPhase(
	ordered []*domain.ServiceSpec,
	env *domain.DeploymentEnv,
	pipelines map[string]*domain.Pipeline,
	state *domain.ReportedState,
	force bool,
) (*domain.PlanPhase, error) {

	phase := &domain.PlanPhase{Name: "build"}

	for _, svc := range ordered {
		deploy := env.FindServiceSpec(svc.Name)
		if deploy == nil {
			continue // service not deployed in this environment
		}

		pipeline, ok := pipelines[svc.Pipeline]
		if !ok {
			return nil, fmt.Errorf("pipeline %q not found for service %q", svc.Pipeline, svc.Name)
		}

		hash := HashServiceConfig(svc, deploy, pipeline)
		changed := force || state.Services[svc.Name] == nil || state.Services[svc.Name].ConfigHash != hash

		stages, err := pipeline.GetStagesFor(deploy.Accepts)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", svc.Name, err)
		}

		var prevStepID string
		for _, stage := range stages {
			stepID := fmt.Sprintf("build-%s-%s", svc.Name, stage.ID)
			step := &domain.PlanStep{
				ID:          stepID,
				Description: fmt.Sprintf("build %s / %s", svc.Name, stage.ID),
				Action: domain.BuildAction{
					Service: svc.Name,
					Stage:   stage.ID,
					Command: stageCommand(stage),
					Args:    stageArgs(stage),
				},
			}
			if prevStepID != "" {
				step.DependsOn = []string{prevStepID}
			}
			if changed {
				step.Status = domain.StepPending
			} else {
				step.Status = domain.StepSkipped
			}
			phase.Steps = append(phase.Steps, step)
			prevStepID = stepID
		}
	}
	return phase, nil
}

// generateProvisionPhase creates provision steps for infrastructure resources.
func generateProvisionPhase(
	env *domain.DeploymentEnv,
	state *domain.ReportedState,
	force bool,
) *domain.PlanPhase {

	phase := &domain.PlanPhase{Name: "provision"}

	for _, dep := range env.Dependencies {
		hash := HashInfraConfig(dep)
		changed := force || state.Infra[dep.Name] == nil || state.Infra[dep.Name].ConfigHash != hash

		step := &domain.PlanStep{
			ID:          fmt.Sprintf("provision-%s", dep.Name),
			Description: fmt.Sprintf("provision %s via %s", dep.Name, dep.Provision.Via),
			Action: domain.ProvisionAction{
				Resource: dep.Name,
				Via:      string(dep.Provision.Via),
				Params:   provisionParams(dep),
			},
		}
		if changed {
			step.Status = domain.StepPending
		} else {
			step.Status = domain.StepSkipped
		}
		phase.Steps = append(phase.Steps, step)
	}
	return phase
}

// generateDeployPhase creates deploy steps in topological order.
func generateDeployPhase(
	ordered []*domain.ServiceSpec,
	env *domain.DeploymentEnv,
	pipelines map[string]*domain.Pipeline,
	state *domain.ReportedState,
	force bool,
) (*domain.PlanPhase, error) {

	phase := &domain.PlanPhase{Name: "deploy"}

	// Build a map of service → binding env for quick lookup.
	bindingMap := make(map[string]map[string]string)
	for _, b := range env.Bindings {
		bindingMap[b.Service] = b.Env
	}

	for _, svc := range ordered {
		deploy := env.FindServiceSpec(svc.Name)
		if deploy == nil {
			continue
		}

		pipeline, ok := pipelines[svc.Pipeline]
		if !ok {
			return nil, fmt.Errorf("pipeline %q not found for service %q", svc.Pipeline, svc.Name)
		}

		hash := HashServiceConfig(svc, deploy, pipeline)
		changed := force || state.Services[svc.Name] == nil || state.Services[svc.Name].ConfigHash != hash

		step := &domain.PlanStep{
			ID:          fmt.Sprintf("deploy-%s", svc.Name),
			Description: fmt.Sprintf("deploy %s (%s)", svc.Name, deploy.Accepts),
			Action: domain.DeployAction{
				Service:     svc.Name,
				ArtifactRef: deploy.Accepts, // will be resolved to actual ref at execution time
				Compute:     deploy.Compute,
				Env:         bindingMap[svc.Name],
			},
		}

		// Deploy depends on the same service's build steps and all dependency services.
		for _, dep := range svc.DependsOn {
			depStepID := fmt.Sprintf("deploy-%s", dep)
			step.DependsOn = append(step.DependsOn, depStepID)
		}

		if changed {
			step.Status = domain.StepPending
		} else {
			step.Status = domain.StepSkipped
		}
		phase.Steps = append(phase.Steps, step)
	}
	return phase, nil
}

// generateNetworkPhase creates network configuration steps.
func generateNetworkPhase(
	env *domain.DeploymentEnv,
	state *domain.ReportedState,
	force bool,
) *domain.PlanPhase {

	phase := &domain.PlanPhase{Name: "network"}

	if env.Network == nil {
		return phase
	}

	for _, ing := range env.Network.Ingress {
		hash := HashIngressConfig(ing)
		changed := force || state.Ingress[ing.Name] == nil || state.Ingress[ing.Name].ConfigHash != hash

		cfg := make(map[string]string)
		if ing.Bind != nil {
			cfg["bindIP"] = ing.Bind.IP
			cfg["httpPort"] = fmt.Sprintf("%d", ing.Bind.HTTP)
			if ing.Bind.HTTPS > 0 {
				cfg["httpsPort"] = fmt.Sprintf("%d", ing.Bind.HTTPS)
			}
		}
		for i, r := range ing.Routes {
			cfg[fmt.Sprintf("route[%d]", i)] = fmt.Sprintf("%s → %s:%d", r.Path, r.Service, r.Port)
		}

		step := &domain.PlanStep{
			ID:          fmt.Sprintf("network-%s", ing.Name),
			Description: fmt.Sprintf("configure %s (%s)", ing.Name, ing.Type),
			Action: domain.NetworkAction{
				Ingress: ing.Name,
				Type:    ing.Type,
				Config:  cfg,
			},
		}
		if changed {
			step.Status = domain.StepPending
		} else {
			step.Status = domain.StepSkipped
		}
		phase.Steps = append(phase.Steps, step)
	}
	return phase
}

// ── Helpers ──────────────────────────────────────────────────────────────

func stageCommand(stage *domain.Stage) string {
	if stage.Action == nil {
		return ""
	}
	return stage.Action.Command
}

func stageArgs(stage *domain.Stage) []string {
	if stage.Action == nil {
		return nil
	}
	return stage.Action.Args
}

func provisionParams(dep *domain.InfraResource) map[string]string {
	params := make(map[string]string)
	p := dep.Provision
	if p == nil {
		return params
	}
	switch p.Via {
	case domain.ProvisionViaDocker:
		params["image"] = p.Image
		for k, v := range p.Env {
			params["env."+k] = v
		}
	case domain.ProvisionViaTerraform:
		params["module"] = p.Module
	case domain.ProvisionViaHelm:
		params["chart"] = p.Chart
	case domain.ProvisionViaExternal:
		params["endpoint"] = p.Endpoint
	}
	return params
}
