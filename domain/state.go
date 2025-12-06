package domain

// State 状态
type State struct {
	ID                string
	Name              string
	Description       string
	ExpectedAssetTypes []*AssetType
	AvailableActions  []*Action
}

// NewState 创建新状态
func NewState(id, name, description string) *State {
	return &State{
		ID:                 id,
		Name:               name,
		Description:        description,
		ExpectedAssetTypes: make([]*AssetType, 0),
		AvailableActions:   make([]*Action, 0),
	}
}

// AddExpectedAssetType 添加期望的资产类型
func (s *State) AddExpectedAssetType(assetType *AssetType) {
	s.ExpectedAssetTypes = append(s.ExpectedAssetTypes, assetType)
}

// AddAvailableAction 添加可用动作
func (s *State) AddAvailableAction(action *Action) {
	s.AvailableActions = append(s.AvailableActions, action)
}

// HasExpectedAssetType 检查是否期望某种资产类型
func (s *State) HasExpectedAssetType(assetType *AssetType) bool {
	for _, at := range s.ExpectedAssetTypes {
		if at.ID == assetType.ID {
			return true
		}
	}
	return false
}

