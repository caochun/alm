package domain

// StateTransitionRule 状态转换规则
type StateTransitionRule struct {
	FromState          *State
	ToState            *State
	RequiredAction     *Action
	Conditions         map[string]interface{} // 条件定义
	InputAssetTypes    []*AssetType            // 输入资产类型（前一个状态生成的资产）
	GeneratedAssetTypes []*AssetType            // 输出资产类型（当前转换会生成的资产）
}

// NewStateTransitionRule 创建新的状态转换规则
func NewStateTransitionRule(from, to *State, requiredAction *Action) *StateTransitionRule {
	return &StateTransitionRule{
		FromState:          from,
		ToState:            to,
		RequiredAction:     requiredAction,
		Conditions:         make(map[string]interface{}),
		InputAssetTypes:    make([]*AssetType, 0),
		GeneratedAssetTypes: make([]*AssetType, 0),
	}
}

// AddCondition 添加转换条件
func (str *StateTransitionRule) AddCondition(key string, value interface{}) {
	str.Conditions[key] = value
}

// AddInputAssetType 添加输入资产类型
func (str *StateTransitionRule) AddInputAssetType(assetType *AssetType) {
	str.InputAssetTypes = append(str.InputAssetTypes, assetType)
}

// AddGeneratedAssetType 添加会生成的资产类型
func (str *StateTransitionRule) AddGeneratedAssetType(assetType *AssetType) {
	str.GeneratedAssetTypes = append(str.GeneratedAssetTypes, assetType)
}

// EvaluateConditions 评估条件是否满足
func (str *StateTransitionRule) EvaluateConditions(context map[string]interface{}) bool {
	// 如果没有定义条件，则默认通过
	if len(str.Conditions) == 0 {
		return true
	}

	// 简单的条件评估：检查context中是否包含所有必需的条件值
	for key, expectedValue := range str.Conditions {
		actualValue, exists := context[key]
		
		// 如果期望值是"required"，只要存在该key即可
		if expectedValueStr, ok := expectedValue.(string); ok && expectedValueStr == "required" {
			if !exists || actualValue == nil || actualValue == "" {
				return false
			}
			continue
		}
		
		// 如果key不存在，验证失败
		if !exists {
			return false
		}
		
		// 简单的相等比较，实际应用中可能需要更复杂的条件评估逻辑
		if actualValue != expectedValue {
			return false
		}
	}

	return true
}

