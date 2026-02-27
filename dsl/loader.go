package dsl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alm/domain"
)

// LoadPipelinesFromDir loads every AppPipeline YAML file in dir and returns
// a map keyed by pipeline name. Non-YAML files and sub-directories are ignored.
func LoadPipelinesFromDir(dir string) (map[string]*domain.Pipeline, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline directory %s: %w", dir, err)
	}

	pipelines := make(map[string]*domain.Pipeline)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		p, err := ParsePipeline(path)
		if err != nil {
			return nil, fmt.Errorf("loading pipeline %s: %w", path, err)
		}
		pipelines[p.Name] = p
	}
	return pipelines, nil
}
