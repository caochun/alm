package dsl

import (
	"fmt"
	"os"

	"github.com/alm/domain"
	"gopkg.in/yaml.v3"
)

// StateMachineDSL 状态机DSL结构
type StateMachineDSL struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	AssetTypes  []AssetTypeDSL  `yaml:"asset_types"`
	Actions     []ActionDSL     `yaml:"actions"`
	States      []StateDSL      `yaml:"states"`
	Transitions []TransitionDSL `yaml:"transitions"`
}

// AssetTypeDSL 资产类型DSL
type AssetTypeDSL struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Schema      map[string]interface{} `yaml:"schema"`
}

// ActionDSL 动作DSL
type ActionDSL struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"` // tool | manual
	Description string                 `yaml:"description"`
	Command     string                 `yaml:"command"`
	Args        []string               `yaml:"args"`
	Parameters  map[string]interface{} `yaml:"parameters"`
}

// StateDSL 状态DSL
type StateDSL struct {
	ID                 string   `yaml:"id"`
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	ExpectedAssetTypes []string `yaml:"expected_asset_types"`
}

// TransitionDSL 转换规则DSL
type TransitionDSL struct {
	From                string                 `yaml:"from"`
	To                  string                 `yaml:"to"`
	Action              string                 `yaml:"action"`
	Conditions          map[string]interface{} `yaml:"conditions"`
	InputAssetTypes     []string               `yaml:"input_asset_types"`
	GeneratedAssetTypes []string               `yaml:"generated_asset_types"`
}

// ParseStateMachine 解析YAML文件并创建状态机模板
func ParseStateMachine(filePath string) (*domain.StateMachineTemplate, map[string]*domain.AssetType, map[string]*domain.Action, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dsl StateMachineDSL
	if err := yaml.Unmarshal(data, &dsl); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 创建资产类型
	assetTypes, err := createAssetTypes(&dsl)
	if err != nil {
		return nil, nil, nil, err
	}

	// 创建动作
	actions, err := createActions(&dsl)
	if err != nil {
		return nil, nil, nil, err
	}

	// 创建状态机模板
	template, err := createStateMachineTemplate(&dsl, assetTypes, actions)
	if err != nil {
		return nil, nil, nil, err
	}

	return template, assetTypes, actions, nil
}

// createAssetTypes 创建资产类型
func createAssetTypes(dsl *StateMachineDSL) (map[string]*domain.AssetType, error) {
	assetTypes := make(map[string]*domain.AssetType)

	for _, atDSL := range dsl.AssetTypes {
		assetType := domain.NewAssetType(atDSL.ID, atDSL.Name, atDSL.Description)

		// 设置schema（转换为JSON字符串）
		if atDSL.Schema != nil {
			schemaBytes, err := yaml.Marshal(atDSL.Schema)
			if err == nil {
				assetType.SetSchema(string(schemaBytes))
			}
		}

		assetTypes[atDSL.ID] = assetType
	}

	return assetTypes, nil
}

// createActions 创建动作
func createActions(dsl *StateMachineDSL) (map[string]*domain.Action, error) {
	actions := make(map[string]*domain.Action)

	for _, actionDSL := range dsl.Actions {
		var actionType domain.ActionType
		switch actionDSL.Type {
		case "tool":
			actionType = domain.ActionTypeTool
		case "manual":
			actionType = domain.ActionTypeManual
		default:
			return nil, fmt.Errorf("unknown action type: %s", actionDSL.Type)
		}

		action := domain.NewAction(actionDSL.ID, actionDSL.Name, actionType, actionDSL.Description)

		if actionDSL.Command != "" {
			action.SetCommand(actionDSL.Command, actionDSL.Args)
		}

		for key, value := range actionDSL.Parameters {
			action.AddParameter(key, value)
		}

		actions[actionDSL.ID] = action
	}

	return actions, nil
}

// createStateMachineTemplate 创建状态机模板
func createStateMachineTemplate(dsl *StateMachineDSL, assetTypes map[string]*domain.AssetType, actions map[string]*domain.Action) (*domain.StateMachineTemplate, error) {
	template := domain.NewStateMachineTemplate(
		dsl.Name,
		dsl.Name,
		dsl.Description,
	)

	// 创建状态映射
	stateMap := make(map[string]*domain.State)

	// 创建状态
	for _, stateDSL := range dsl.States {
		state := domain.NewState(stateDSL.ID, stateDSL.Name, stateDSL.Description)

		// 添加期望的资产类型（如果定义了）
		if len(stateDSL.ExpectedAssetTypes) > 0 {
			for _, assetTypeID := range stateDSL.ExpectedAssetTypes {
				if assetType, ok := assetTypes[assetTypeID]; ok {
					state.AddExpectedAssetType(assetType)
				}
			}
		}

		stateMap[stateDSL.ID] = state
		template.AddState(state)
	}

	// 创建转换规则
	for _, transDSL := range dsl.Transitions {
		fromState, ok := stateMap[transDSL.From]
		if !ok {
			return nil, fmt.Errorf("state not found: %s", transDSL.From)
		}

		toState, ok := stateMap[transDSL.To]
		if !ok {
			return nil, fmt.Errorf("state not found: %s", transDSL.To)
		}

		action, ok := actions[transDSL.Action]
		if !ok {
			return nil, fmt.Errorf("action not found: %s", transDSL.Action)
		}

		// 将action添加到fromState的AvailableActions中
		// 检查是否已经添加过（避免重复）
		actionExists := false
		for _, existingAction := range fromState.AvailableActions {
			if existingAction.ID == action.ID {
				actionExists = true
				break
			}
		}
		if !actionExists {
			fromState.AddAvailableAction(action)
		}

		rule := domain.NewStateTransitionRule(fromState, toState, action)

		// 添加条件
		for key, value := range transDSL.Conditions {
			rule.AddCondition(key, value)
		}

		// 添加输入资产类型
		for _, assetTypeID := range transDSL.InputAssetTypes {
			if assetType, ok := assetTypes[assetTypeID]; ok {
				rule.AddInputAssetType(assetType)
			}
		}

		// 添加输出资产类型
		for _, assetTypeID := range transDSL.GeneratedAssetTypes {
			if assetType, ok := assetTypes[assetTypeID]; ok {
				rule.AddGeneratedAssetType(assetType)
			}
		}

		template.AddTransitionRule(rule)
	}

	return template, nil
}
