package domain

import "time"

// ExecutionPlan represents a complete plan for building, provisioning, deploying,
// and configuring an application in a specific environment. Plans are generated
// from DSL definitions and can be reviewed before execution.
type ExecutionPlan struct {
	App         string       `json:"app"`
	Environment string       `json:"environment"`
	Phases      []*PlanPhase `json:"phases"`
	CreatedAt   time.Time    `json:"createdAt"`
}

// PlanPhase groups steps that belong to the same execution stage.
type PlanPhase struct {
	Name  string      `json:"name"` // "build" | "provision" | "deploy" | "network"
	Steps []*PlanStep `json:"steps"`
}

// PlanStep is a single executable unit within a phase.
type PlanStep struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Action      StepAction `json:"action"`
	DependsOn   []string   `json:"dependsOn,omitempty"` // step IDs within the same phase
	Status      StepStatus `json:"status"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// StepStatus tracks the lifecycle of a plan step.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepSucceeded StepStatus = "succeeded"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

// StepAction is the interface implemented by all action types.
type StepAction interface {
	ActionKind() string
}

// BuildAction runs a pipeline stage to produce a build artifact.
type BuildAction struct {
	Service string   `json:"service"`
	Stage   string   `json:"stage"`   // pipeline stage ID
	Command string   `json:"command"` // tool command to execute
	Args    []string `json:"args,omitempty"`
}

func (BuildAction) ActionKind() string { return "build" }

// ProvisionAction provisions an infrastructure resource.
type ProvisionAction struct {
	Resource string            `json:"resource"`
	Via      string            `json:"via"` // docker | terraform | helm | external
	Params   map[string]string `json:"params,omitempty"`
}

func (ProvisionAction) ActionKind() string { return "provision" }

// DeployAction deploys a service with a specific artifact and configuration.
type DeployAction struct {
	Service     string            `json:"service"`
	ArtifactRef string            `json:"artifactRef"`
	Compute     *ComputeSpec      `json:"compute"`
	Env         map[string]string `json:"env,omitempty"` // interpolated bindings
}

func (DeployAction) ActionKind() string { return "deploy" }

// NetworkAction configures an ingress / network resource.
type NetworkAction struct {
	Ingress string            `json:"ingress"`
	Type    string            `json:"type"` // nginx | traefik | haproxy
	Config  map[string]string `json:"config,omitempty"`
}

func (NetworkAction) ActionKind() string { return "network" }

// CountByStatus returns the number of steps with each status across all phases.
func (p *ExecutionPlan) CountByStatus() map[StepStatus]int {
	counts := make(map[StepStatus]int)
	for _, phase := range p.Phases {
		for _, step := range phase.Steps {
			counts[step.Status]++
		}
	}
	return counts
}
