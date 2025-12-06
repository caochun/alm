package domain

// AssetType 资产类型
type AssetType struct {
	ID             string
	Name           string
	Description    string
	Schema         string // JSON Schema或其他格式的schema
	ValidationRules map[string]interface{}
}

// NewAssetType 创建新的资产类型
func NewAssetType(id, name, description string) *AssetType {
	return &AssetType{
		ID:              id,
		Name:            name,
		Description:     description,
		ValidationRules: make(map[string]interface{}),
	}
}

// SetSchema 设置schema
func (at *AssetType) SetSchema(schema string) {
	at.Schema = schema
}

// AddValidationRule 添加验证规则
func (at *AssetType) AddValidationRule(key string, value interface{}) {
	at.ValidationRules[key] = value
}

// Validate 验证资产是否符合此类型
func (at *AssetType) Validate(asset *ConcreteAsset) bool {
	// TODO: 实现基于schema和验证规则的验证逻辑
	return true
}

