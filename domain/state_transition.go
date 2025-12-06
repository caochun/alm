package domain

import "time"

// StateTransition 状态转换记录
type StateTransition struct {
	ID            string
	SoftwareAsset *SoftwareAsset
	FromState     *State
	ToState       *State
	Action        *Action
	Conditions    map[string]interface{}
	ActionResult  *ActionExecutionResult
	Timestamp     time.Time
	Operator      string // 操作人
}

// NewStateTransition 创建新的状态转换记录
func NewStateTransition(id string, softwareAsset *SoftwareAsset, from, to *State, action *Action, conditions map[string]interface{}, operator string) *StateTransition {
	return &StateTransition{
		ID:            id,
		SoftwareAsset: softwareAsset,
		FromState:     from,
		ToState:       to,
		Action:        action,
		Conditions:    conditions,
		Timestamp:     time.Now(),
		Operator:      operator,
	}
}

