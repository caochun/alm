package domain

import "time"

// ReportedState records the actual deployed state of an application in an
// environment. It is persisted to disk and compared against DSL definitions
// to enable incremental planning — only changed components need re-execution.
type ReportedState struct {
	App         string                    `yaml:"app"         json:"app"`
	Environment string                    `yaml:"environment" json:"environment"`
	Services    map[string]*ServiceState  `yaml:"services"    json:"services"`
	Infra       map[string]*InfraState    `yaml:"infra"       json:"infra"`
	Ingress     map[string]*IngressState  `yaml:"ingress"     json:"ingress"`
	UpdatedAt   time.Time                 `yaml:"updatedAt"   json:"updatedAt"`
}

// ServiceState captures the deployed state of a single service.
type ServiceState struct {
	ConfigHash  string    `yaml:"configHash"  json:"configHash"`  // SHA-256 of relevant DSL config
	ArtifactRef string    `yaml:"artifactRef" json:"artifactRef"` // e.g., docker image tag or jar path
	Status      string    `yaml:"status"      json:"status"`      // running | stopped | failed
	DeployedAt  time.Time `yaml:"deployedAt"  json:"deployedAt"`
}

// InfraState captures the provisioned state of an infrastructure resource.
type InfraState struct {
	ConfigHash    string            `yaml:"configHash"    json:"configHash"`
	Status        string            `yaml:"status"        json:"status"` // running | stopped | failed
	Outputs       map[string]string `yaml:"outputs"       json:"outputs"` // runtime values (host, port, etc.)
	ProvisionedAt time.Time         `yaml:"provisionedAt" json:"provisionedAt"`
}

// IngressState captures the configured state of an ingress resource.
type IngressState struct {
	ConfigHash   string    `yaml:"configHash"   json:"configHash"`
	Status       string    `yaml:"status"       json:"status"`
	ConfiguredAt time.Time `yaml:"configuredAt" json:"configuredAt"`
}

// NewReportedState returns an empty ReportedState for the given app and environment.
func NewReportedState(app, env string) *ReportedState {
	return &ReportedState{
		App:         app,
		Environment: env,
		Services:    make(map[string]*ServiceState),
		Infra:       make(map[string]*InfraState),
		Ingress:     make(map[string]*IngressState),
	}
}
