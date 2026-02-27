package domain

import "fmt"

// Pipeline defines an application's asset transformation pipeline.
// It is a DAG of stages where each stage transforms one asset type into another.
// This is the app developer's concern: how to go from source code to a deployable artifact.
type Pipeline struct {
	Name         string
	Description  string
	Stages       []*Stage
	Deliverables []string // asset type IDs this pipeline can produce (possible exit points)
}

// Stage is a single transformation step in the pipeline.
type Stage struct {
	ID       string
	Name     string
	Requires []string // input asset type IDs (empty for the first stage)
	Produces string   // output asset type ID
	Action   *PipelineAction
}

// PipelineAction defines what tool or command to execute for a stage.
type PipelineAction struct {
	Type       ActionType
	Command    string
	Args       []string
	Parameters map[string]interface{}
}

// ActionType categorises how an action is executed.
type ActionType string

const (
	ActionTypeTool   ActionType = "tool"   // executes an external CLI command
	ActionTypeManual ActionType = "manual" // requires human intervention
	ActionTypeHTTP   ActionType = "http"   // calls an HTTP endpoint
)

// CanProduce reports whether this pipeline can produce the given asset type.
func (p *Pipeline) CanProduce(assetType string) bool {
	for _, d := range p.Deliverables {
		if d == assetType {
			return true
		}
	}
	return false
}

// GetStagesFor returns the ordered stages needed to produce targetAssetType.
// It performs a DFS post-order traversal backwards through the DAG, collecting
// only the stages required to reach the target — skipping unneeded branches.
func (p *Pipeline) GetStagesFor(targetAssetType string) ([]*Stage, error) {
	if !p.CanProduce(targetAssetType) {
		return nil, fmt.Errorf("pipeline %q cannot produce %q (deliverables: %v)",
			p.Name, targetAssetType, p.Deliverables)
	}

	// Index: produced asset type → stage that produces it.
	stageByProduct := make(map[string]*Stage, len(p.Stages))
	for _, s := range p.Stages {
		stageByProduct[s.Produces] = s
	}

	visited := make(map[string]bool)
	result := make([]*Stage, 0)

	var collect func(assetType string)
	collect = func(assetType string) {
		if visited[assetType] {
			return
		}
		visited[assetType] = true

		stage, ok := stageByProduct[assetType]
		if !ok {
			return // no stage produces this; it is an external input (e.g. git repository URL)
		}

		// Depth-first: collect all required stages before appending this one,
		// which guarantees topological (dependency-first) ordering in result.
		for _, req := range stage.Requires {
			collect(req)
		}
		result = append(result, stage)
	}

	collect(targetAssetType)
	return result, nil
}
