package domain

import "time"

// ConcreteAsset 具体资产
type ConcreteAsset struct {
	ID                   string
	Name                 string
	AssetType            *AssetType
	Location             string // 存储位置（路径、URL等）
	Version              string
	Metadata             map[string]interface{}
	GeneratedAtState     *State
	GeneratedByTransition *StateTransition
	InputAssets          []*ConcreteAsset // 依赖的输入资产（用于追溯）
	CreatedAt            time.Time
	SoftwareAsset        *SoftwareAsset
}

// NewConcreteAsset 创建新的具体资产
func NewConcreteAsset(id, name string, assetType *AssetType, location, version string) *ConcreteAsset {
	return &ConcreteAsset{
		ID:        id,
		Name:      name,
		AssetType: assetType,
		Location:  location,
		Version:   version,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

// SetGeneratedContext 设置生成上下文
func (ca *ConcreteAsset) SetGeneratedContext(state *State, transition *StateTransition) {
	ca.GeneratedAtState = state
	ca.GeneratedByTransition = transition
}

// AddInputAsset 添加输入资产依赖
func (ca *ConcreteAsset) AddInputAsset(inputAsset *ConcreteAsset) {
	ca.InputAssets = append(ca.InputAssets, inputAsset)
}

// AddMetadata 添加元数据
func (ca *ConcreteAsset) AddMetadata(key string, value interface{}) {
	ca.Metadata[key] = value
}

// GetMetadata 获取元数据
func (ca *ConcreteAsset) GetMetadata(key string) (interface{}, bool) {
	value, exists := ca.Metadata[key]
	return value, exists
}

