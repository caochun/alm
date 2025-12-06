package domain

// StateMachineTemplate 状态机模板
type StateMachineTemplate struct {
	ID             string
	Name           string
	Description    string
	States         []*State
	TransitionRules []*StateTransitionRule
}

// NewStateMachineTemplate 创建新的状态机模板
func NewStateMachineTemplate(id, name, description string) *StateMachineTemplate {
	return &StateMachineTemplate{
		ID:              id,
		Name:            name,
		Description:     description,
		States:          make([]*State, 0),
		TransitionRules: make([]*StateTransitionRule, 0),
	}
}

// AddState 添加状态
func (smt *StateMachineTemplate) AddState(state *State) {
	smt.States = append(smt.States, state)
}

// AddTransitionRule 添加转换规则
func (smt *StateMachineTemplate) AddTransitionRule(rule *StateTransitionRule) {
	smt.TransitionRules = append(smt.TransitionRules, rule)
}

// ValidateTransition 验证状态转换是否合法
func (smt *StateMachineTemplate) ValidateTransition(from, to *State, conditions map[string]interface{}) bool {
	// 检查状态是否在模板中
	fromExists := false
	toExists := false
	for _, state := range smt.States {
		if state.ID == from.ID {
			fromExists = true
		}
		if state.ID == to.ID {
			toExists = true
		}
	}
	if !fromExists || !toExists {
		return false
	}

	// 查找转换规则
	rule := smt.FindTransitionRule(from, to)
	if rule == nil {
		return false
	}

	// 评估条件
	return rule.EvaluateConditions(conditions)
}

// FindTransitionRule 查找转换规则
func (smt *StateMachineTemplate) FindTransitionRule(from, to *State) *StateTransitionRule {
	for _, rule := range smt.TransitionRules {
		if rule.FromState.ID == from.ID && rule.ToState.ID == to.ID {
			return rule
		}
	}
	return nil
}

// GetAvailableTransitions 获取当前状态可用的转换
func (smt *StateMachineTemplate) GetAvailableTransitions(currentState *State) []*StateTransitionRule {
	result := make([]*StateTransitionRule, 0)
	for _, rule := range smt.TransitionRules {
		if rule.FromState.ID == currentState.ID {
			result = append(result, rule)
		}
	}
	return result
}

