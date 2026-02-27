package dsl

import (
	"fmt"
	"os"

	"github.com/alm/domain"
	"gopkg.in/yaml.v3"
)

// pipelineDSL mirrors the AppPipeline YAML structure.
type pipelineDSL struct {
	Kind         string       `yaml:"kind"`
	Name         string       `yaml:"name"`
	Description  string       `yaml:"description"`
	Stages       []stageDSL   `yaml:"stages"`
	Deliverables []string     `yaml:"deliverables"`
}

type stageDSL struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Requires []string  `yaml:"requires"`
	Produces string    `yaml:"produces"`
	Action   actionDSL `yaml:"action"`
}

type actionDSL struct {
	Type       string                 `yaml:"type"`
	Command    string                 `yaml:"command"`
	Args       []string               `yaml:"args"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// ParsePipeline reads an AppPipeline YAML file and returns a domain.Pipeline.
func ParsePipeline(path string) (*domain.Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline file %s: %w", path, err)
	}
	return parsePipelineBytes(data, path)
}

func parsePipelineBytes(data []byte, source string) (*domain.Pipeline, error) {
	var d pipelineDSL
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing pipeline YAML %s: %w", source, err)
	}

	if d.Kind != "AppPipeline" {
		return nil, fmt.Errorf("%s: expected kind AppPipeline, got %q", source, d.Kind)
	}
	if d.Name == "" {
		return nil, fmt.Errorf("%s: pipeline name is required", source)
	}
	if len(d.Deliverables) == 0 {
		return nil, fmt.Errorf("%s: pipeline must declare at least one deliverable", source)
	}

	// Ensure every declared deliverable is produced by some stage.
	producedBy := make(map[string]bool, len(d.Stages))
	for _, s := range d.Stages {
		producedBy[s.Produces] = true
	}
	for _, del := range d.Deliverables {
		if !producedBy[del] {
			return nil, fmt.Errorf("%s: deliverable %q is not produced by any stage", source, del)
		}
	}

	pipeline := &domain.Pipeline{
		Name:         d.Name,
		Description:  d.Description,
		Deliverables: d.Deliverables,
		Stages:       make([]*domain.Stage, 0, len(d.Stages)),
	}

	for i, s := range d.Stages {
		if s.ID == "" {
			return nil, fmt.Errorf("%s: stage[%d] id is required", source, i)
		}
		if s.Produces == "" {
			return nil, fmt.Errorf("%s: stage %q must declare what it produces", source, s.ID)
		}
		pipeline.Stages = append(pipeline.Stages, &domain.Stage{
			ID:       s.ID,
			Name:     s.Name,
			Requires: s.Requires,
			Produces: s.Produces,
			Action: &domain.PipelineAction{
				Type:       domain.ActionType(s.Action.Type),
				Command:    s.Action.Command,
				Args:       s.Action.Args,
				Parameters: s.Action.Parameters,
			},
		})
	}

	return pipeline, nil
}
