package dsl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alm/domain"
	"gopkg.in/yaml.v3"
)

// appArchDSL mirrors the AppArchitecture YAML structure.
type appArchDSL struct {
	Kind        string           `yaml:"kind"`
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Services    []serviceSpecDSL `yaml:"services"`
}

type serviceSpecDSL struct {
	Name       string   `yaml:"name"`
	Pipeline   string   `yaml:"pipeline"`
	Repository string   `yaml:"repository"`
	DependsOn  []string `yaml:"depends_on"`
}

// ParseAppArchitecture reads an AppArchitecture YAML file and returns a domain.AppArchitecture.
func ParseAppArchitecture(path string) (*domain.AppArchitecture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading app architecture file %s: %w", path, err)
	}
	return parseAppArchBytes(data, path)
}

func parseAppArchBytes(data []byte, source string) (*domain.AppArchitecture, error) {
	var d appArchDSL
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing app architecture YAML %s: %w", source, err)
	}

	if d.Kind != "AppArchitecture" {
		return nil, fmt.Errorf("%s: expected kind AppArchitecture, got %q", source, d.Kind)
	}
	if d.Name == "" {
		return nil, fmt.Errorf("%s: app architecture name is required", source)
	}
	if len(d.Services) == 0 {
		return nil, fmt.Errorf("%s: app architecture must define at least one service", source)
	}

	// Validate: unique names and all depends_on references exist.
	known := make(map[string]bool, len(d.Services))
	for _, s := range d.Services {
		if s.Name == "" {
			return nil, fmt.Errorf("%s: service name is required", source)
		}
		if s.Pipeline == "" {
			return nil, fmt.Errorf("%s: service %q must reference a pipeline", source, s.Name)
		}
		if known[s.Name] {
			return nil, fmt.Errorf("%s: duplicate service name %q", source, s.Name)
		}
		known[s.Name] = true
	}
	for _, s := range d.Services {
		for _, dep := range s.DependsOn {
			if !known[dep] {
				return nil, fmt.Errorf("%s: service %q depends on unknown service %q", source, s.Name, dep)
			}
		}
	}

	arch := &domain.AppArchitecture{
		Name:        d.Name,
		Description: d.Description,
		Services:    make([]*domain.ServiceSpec, 0, len(d.Services)),
	}
	for _, s := range d.Services {
		arch.Services = append(arch.Services, &domain.ServiceSpec{
			Name:       s.Name,
			Pipeline:   s.Pipeline,
			Repository: s.Repository,
			DependsOn:  s.DependsOn,
		})
	}

	// Validate no cyclic dependencies.
	if _, err := arch.TopologicalOrder(); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}

	return arch, nil
}

// WriteAppArchitecture serializes a domain.AppArchitecture to YAML and writes
// it to the given path. The output is validated via round-trip parsing.
func WriteAppArchitecture(path string, arch *domain.AppArchitecture) error {
	d := appArchDSL{
		Kind:        "AppArchitecture",
		Name:        arch.Name,
		Description: arch.Description,
	}
	for _, svc := range arch.Services {
		d.Services = append(d.Services, serviceSpecDSL{
			Name:       svc.Name,
			Pipeline:   svc.Pipeline,
			Repository: svc.Repository,
			DependsOn:  svc.DependsOn,
		})
	}

	data, err := yaml.Marshal(&d)
	if err != nil {
		return fmt.Errorf("marshal app architecture: %w", err)
	}

	if _, err := parseAppArchBytes(data, path); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
