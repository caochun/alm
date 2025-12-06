package domain

import (
	"time"
)

// SoftwareAsset 软件资产聚合根
type SoftwareAsset struct {
	ID                   string
	Name                 string
	Description          string
	CurrentState         *State
	StateMachineTemplate *StateMachineTemplate
	ConcreteAssets       []*ConcreteAsset
	TransitionHistory    []*StateTransition
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// NewSoftwareAsset 创建新的软件资产
func NewSoftwareAsset(id, name, description string, template *StateMachineTemplate) (*SoftwareAsset, error) {
	if template == nil {
		return nil, ErrInvalidStateMachineTemplate
	}
	if len(template.States) == 0 {
		return nil, ErrEmptyStates
	}

	// 设置初始状态为模板的第一个状态
	initialState := template.States[0]

	now := time.Now()
	return &SoftwareAsset{
		ID:                   id,
		Name:                 name,
		Description:          description,
		CurrentState:         initialState,
		StateMachineTemplate: template,
		ConcreteAssets:       make([]*ConcreteAsset, 0),
		TransitionHistory:    make([]*StateTransition, 0),
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// TransitionTo 转换到目标状态
func (sa *SoftwareAsset) TransitionTo(targetState *State, action *Action, conditions map[string]interface{}, operator string) (*StateTransition, error) {
	// 验证转换是否合法
	if !sa.StateMachineTemplate.ValidateTransition(sa.CurrentState, targetState, conditions) {
		return nil, ErrInvalidTransition
	}

	// 查找对应的转换规则
	rule := sa.StateMachineTemplate.FindTransitionRule(sa.CurrentState, targetState)
	if rule == nil {
		return nil, ErrTransitionRuleNotFound
	}

	// 验证动作是否匹配
	if rule.RequiredAction != nil && action.ID != rule.RequiredAction.ID {
		return nil, ErrActionMismatch
	}

	// 查找输入资产（前一个状态生成的资产）
	inputAssets := sa.findInputAssets(rule.InputAssetTypes)

	// 创建状态转换记录
	transition := &StateTransition{
		ID:            GenerateID(),
		SoftwareAsset: sa,
		FromState:     sa.CurrentState,
		ToState:       targetState,
		Action:        action,
		Conditions:    conditions,
		Timestamp:     time.Now(),
		Operator:      operator,
	}

	// 执行动作（如果是工具动作）
	if action != nil && action.Type == ActionTypeTool {
		// 构建执行上下文，包含输入资产信息
		execContext := map[string]interface{}{
			"softwareAsset": sa,
			"fromState":     sa.CurrentState,
			"toState":       targetState,
			"conditions":    conditions,
			"inputAssets":   inputAssets, // 传递输入资产
		}
		
		// 将输入资产的位置信息添加到上下文
		if len(inputAssets) > 0 {
			// 第一个输入资产通常作为主要输入
			execContext["sourceDir"] = inputAssets[0].Location
			execContext["outputDir"] = inputAssets[0].Location
			execContext["jarPath"] = inputAssets[0].Location
			
			// 根据资产类型设置对应的变量
			for _, asset := range inputAssets {
				if asset.AssetType != nil {
					switch asset.AssetType.ID {
					case "source-code":
						execContext["sourceDir"] = asset.Location
					case "jar-file":
						execContext["jarPath"] = asset.Location
					}
				}
			}
		}
		
		// 合并conditions到execContext
		for k, v := range conditions {
			execContext[k] = v
		}
		
		result := action.Execute(execContext)
		transition.ActionResult = result
		
		// 将输入资产信息保存到结果metadata中，便于后续提取
		if len(inputAssets) > 0 {
			result.Metadata["sourceDir"] = inputAssets[0].Location
			if len(inputAssets) > 0 && inputAssets[0].AssetType != nil {
				if inputAssets[0].AssetType.ID == "jar-file" {
					result.Metadata["jarPath"] = inputAssets[0].Location
				}
			}
		}

		// 如果执行成功，根据规则创建输出资产
		if result.IsSuccess() && len(rule.GeneratedAssetTypes) > 0 {
			sa.createOutputAssets(rule.GeneratedAssetTypes, targetState, transition, inputAssets, result)
		}
	}

	// 更新状态
	sa.CurrentState = targetState
	sa.TransitionHistory = append(sa.TransitionHistory, transition)
	sa.UpdatedAt = time.Now()

	return transition, nil
}

// findInputAssets 查找输入资产（根据资产类型从当前状态生成的资产中查找）
func (sa *SoftwareAsset) findInputAssets(inputAssetTypes []*AssetType) []*ConcreteAsset {
	if len(inputAssetTypes) == 0 {
		return nil
	}

	inputAssets := make([]*ConcreteAsset, 0)
	// 从当前状态生成的资产中查找
	currentStateAssets := sa.GetConcreteAssetsByState(sa.CurrentState)
	
	for _, assetType := range inputAssetTypes {
		for _, asset := range currentStateAssets {
			if asset.AssetType.ID == assetType.ID {
				inputAssets = append(inputAssets, asset)
				break // 每种类型只取一个（可以根据需要调整）
			}
		}
	}
	
	return inputAssets
}

// createOutputAssets 创建输出资产
func (sa *SoftwareAsset) createOutputAssets(assetTypes []*AssetType, state *State, transition *StateTransition, inputAssets []*ConcreteAsset, result *ActionExecutionResult) {
	// 从执行结果中提取生成的资产信息
	// 这里需要根据实际的工具输出解析资产位置等信息
	// 暂时创建占位资产，实际实现时需要解析工具输出
	
	for _, assetType := range assetTypes {
		// 从执行结果的元数据中提取资产信息（需要根据具体工具实现）
		location := sa.extractAssetLocation(assetType, result)
		
		outputAsset := NewConcreteAsset(
			GenerateID(),
			assetType.Name+"-"+sa.Name,
			assetType,
			location,
			"", // version可以从metadata中提取
		)
		outputAsset.SetGeneratedContext(state, transition)
		
		// 设置输入资产依赖
		for _, inputAsset := range inputAssets {
			outputAsset.AddInputAsset(inputAsset)
		}
		
		// 添加执行结果元数据
		outputAsset.AddMetadata("executionResultId", result.ID)
		outputAsset.AddMetadata("actionId", transition.Action.ID)
		
		sa.AddConcreteAsset(outputAsset)
	}
}

// extractAssetLocation 从执行结果中提取资产位置
// 这个方法需要根据具体工具的输出格式实现
func (sa *SoftwareAsset) extractAssetLocation(assetType *AssetType, result *ActionExecutionResult) string {
	// 从result.Metadata或result.Output中解析资产位置
	// 例如：Maven构建的jar文件路径、Terraform创建的容器ID等
	
	// 首先检查metadata中是否有预设的位置
	if location, ok := result.Metadata["assetLocation"]; ok {
		if loc, ok := location.(string); ok {
			return loc
		}
	}
	
	// 根据资产类型和动作类型推断位置
	if result.Action != nil {
		switch result.Action.ID {
		case "git-clone":
			// Git clone的输出通常是目录
			if _, ok := result.Metadata["repository"]; ok {
				// 从repository URL提取目录名
				// 简化处理：实际应该解析git输出
				return "/workspace/spring-petclinic" // 占位值
			}
		case "maven-build":
			// Maven构建的jar通常在target目录
			if sourceDir, ok := result.Metadata["sourceDir"]; ok {
				if dir, ok := sourceDir.(string); ok {
					return dir + "/target/petclinic.jar" // 占位值
				}
			}
		case "terraform-deploy":
			// Terraform部署的容器ID
			if containerId, ok := result.Metadata["containerId"]; ok {
				if id, ok := containerId.(string); ok {
					return id
				}
			}
			return "container-petclinic-001" // 占位值
		}
	}
	
	return "" // 需要根据实际工具输出实现解析逻辑
}

// AddConcreteAsset 添加具体资产
func (sa *SoftwareAsset) AddConcreteAsset(asset *ConcreteAsset) {
	asset.SoftwareAsset = sa
	sa.ConcreteAssets = append(sa.ConcreteAssets, asset)
	sa.UpdatedAt = time.Now()
}

// GetConcreteAssetsByState 获取指定状态下生成的具体资产
func (sa *SoftwareAsset) GetConcreteAssetsByState(state *State) []*ConcreteAsset {
	result := make([]*ConcreteAsset, 0)
	for _, asset := range sa.ConcreteAssets {
		if asset.GeneratedAtState != nil && asset.GeneratedAtState.ID == state.ID {
			result = append(result, asset)
		}
	}
	return result
}

// GetConcreteAssetsByType 根据资产类型获取具体资产
func (sa *SoftwareAsset) GetConcreteAssetsByType(assetType *AssetType) []*ConcreteAsset {
	result := make([]*ConcreteAsset, 0)
	for _, asset := range sa.ConcreteAssets {
		if asset.AssetType.ID == assetType.ID {
			result = append(result, asset)
		}
	}
	return result
}

// GetAllConcreteAssets 获取所有具体资产
func (sa *SoftwareAsset) GetAllConcreteAssets() []*ConcreteAsset {
	return sa.ConcreteAssets
}

// FindConcreteAssetByID 根据ID查找具体资产
func (sa *SoftwareAsset) FindConcreteAssetByID(id string) *ConcreteAsset {
	for _, asset := range sa.ConcreteAssets {
		if asset.ID == id {
			return asset
		}
	}
	return nil
}
