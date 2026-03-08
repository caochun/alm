package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alm/domain"
	"gopkg.in/yaml.v3"
)

// StateManager handles loading and saving ReportedState to disk.
// State files are stored at workspace/{app}/.alm-state-{env}.yaml.
type StateManager struct {
	WorkspaceRoot string
}

// NewStateManager creates a StateManager rooted at the given workspace directory.
func NewStateManager(workspaceRoot string) *StateManager {
	return &StateManager{WorkspaceRoot: workspaceRoot}
}

func (m *StateManager) statePath(app, env string) string {
	return filepath.Join(m.WorkspaceRoot, app, fmt.Sprintf(".alm-state-%s.yaml", env))
}

// Load reads the ReportedState for the given app and environment.
// Returns a fresh empty state if the file does not exist.
func (m *StateManager) Load(app, env string) (*domain.ReportedState, error) {
	path := m.statePath(app, env)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewReportedState(app, env), nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state domain.ReportedState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state file %s: %w", path, err)
	}

	// Ensure maps are initialized
	if state.Services == nil {
		state.Services = make(map[string]*domain.ServiceState)
	}
	if state.Infra == nil {
		state.Infra = make(map[string]*domain.InfraState)
	}
	if state.Ingress == nil {
		state.Ingress = make(map[string]*domain.IngressState)
	}
	return &state, nil
}

// Save writes the ReportedState to disk, updating the timestamp.
func (m *StateManager) Save(state *domain.ReportedState) error {
	state.UpdatedAt = time.Now()
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	path := m.statePath(state.App, state.Environment)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}
