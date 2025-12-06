package engine

import (
	"fmt"

	"github.com/alm/domain"
	"github.com/alm/internal/manager"
)

// StateMachineEngine 状态机引擎
type StateMachineEngine struct {
	assetManager    AssetManagerInterface
	executorFactory ExecutorFactory
}

// AssetManagerInterface 资产管理器接口
type AssetManagerInterface interface {
	GetAsset() *domain.SoftwareAsset
	GetWorkspacePath() string
	GetConfig() interface{}
	GetTemplate() *domain.StateMachineTemplate
	GetActions() map[string]*domain.Action
}

// ExecutorFactory 执行器工厂接口
type ExecutorFactory interface {
	CreateExecutor(actionID string) (Executor, error)
}

// Executor 执行器接口
type Executor interface {
	Execute(context *ExecutionContext) (*ExecutionResult, error)
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Asset         *domain.SoftwareAsset
	FromState     *domain.State
	ToState       *domain.State
	Action        *domain.Action
	Conditions    map[string]interface{}
	InputAssets   []*domain.ConcreteAsset
	WorkspacePath string
	Config        interface{}
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success        bool
	Output         string
	ErrorMessage   string
	Metadata       map[string]interface{}
	AssetLocations map[string]string // 生成的资产位置映射（资产类型ID -> 位置）
}

// NewStateMachineEngine 创建状态机引擎
func NewStateMachineEngine(assetManager AssetManagerInterface, executorFactory ExecutorFactory) *StateMachineEngine {
	return &StateMachineEngine{
		assetManager:    assetManager,
		executorFactory: executorFactory,
	}
}

// TransitionTo 转换到目标状态
func (e *StateMachineEngine) TransitionTo(targetStateID string, conditions map[string]interface{}, operator string) (*domain.StateTransition, error) {
	asset := e.assetManager.GetAsset()
	template := e.assetManager.GetTemplate()

	// 查找目标状态
	var targetState *domain.State
	for _, state := range template.States {
		if state.ID == targetStateID {
			targetState = state
			break
		}
	}
	if targetState == nil {
		return nil, fmt.Errorf("target state not found: %s", targetStateID)
	}

	// 验证转换是否合法
	if !template.ValidateTransition(asset.CurrentState, targetState, conditions) {
		return nil, fmt.Errorf("invalid transition from %s to %s", asset.CurrentState.ID, targetStateID)
	}

	// 查找转换规则
	rule := template.FindTransitionRule(asset.CurrentState, targetState)
	if rule == nil {
		return nil, fmt.Errorf("transition rule not found")
	}

	// 获取动作
	action := rule.RequiredAction
	if action == nil {
		return nil, fmt.Errorf("no action required for this transition")
	}

	// 查找输入资产
	inputAssets := e.findInputAssets(asset, rule.InputAssetTypes)

	// 创建执行上下文
	execContext := &ExecutionContext{
		Asset:         asset,
		FromState:     asset.CurrentState,
		ToState:       targetState,
		Action:        action,
		Conditions:    conditions,
		InputAssets:   inputAssets,
		WorkspacePath: e.assetManager.GetWorkspacePath(),
		Config:        e.assetManager.GetConfig(),
	}

	// 合并应用配置到conditions（如果需要）
	// 配置已经在execContext.Config中，executor可以访问

	// 执行动作（如果是工具动作）
	var execResult *ExecutionResult
	if action.Type == domain.ActionTypeTool {
		executor, err := e.executorFactory.CreateExecutor(action.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create executor: %w", err)
		}

		execResult, err = executor.Execute(execContext)
		if err != nil {
			return nil, fmt.Errorf("execution failed: %w", err)
		}
	} else {
		// 人为动作，创建成功结果
		execResult = &ExecutionResult{
			Success:  true,
			Metadata: make(map[string]interface{}),
		}
	}

	// 创建状态转换记录
	transition := &domain.StateTransition{
		ID:            domain.GenerateID(),
		SoftwareAsset: asset,
		FromState:     asset.CurrentState,
		ToState:       targetState,
		Action:        action,
		Conditions:    conditions,
		Timestamp:     domain.GetCurrentTime(),
		Operator:      operator,
	}

	// 创建动作执行结果
	if execResult != nil {
		var status domain.ExecutionStatus
		if execResult.Success {
			status = domain.ExecutionStatusSuccess
		} else {
			status = domain.ExecutionStatusFailed
		}

		actionResult := &domain.ActionExecutionResult{
			ID:           domain.GenerateID(),
			Action:       action,
			Status:       status,
			Output:       execResult.Output,
			ErrorMessage: execResult.ErrorMessage,
			Metadata:     execResult.Metadata,
		}
		transition.ActionResult = actionResult
	}

	// 如果执行成功，创建输出资产
	if execResult != nil && execResult.Success && len(rule.GeneratedAssetTypes) > 0 {
		e.createOutputAssets(asset, rule.GeneratedAssetTypes, targetState, transition, inputAssets, execResult)
	}

	// 更新资产状态
	asset.CurrentState = targetState
	asset.TransitionHistory = append(asset.TransitionHistory, transition)
	asset.UpdatedAt = domain.GetCurrentTime()

	// 保存资产状态到持久化存储
	if assetManager, ok := e.assetManager.(*manager.AssetManager); ok {
		if err := assetManager.SaveAssetState(); err != nil {
			// 记录错误但不影响状态转换
			// TODO: 添加日志记录
		}
	}

	return transition, nil
}

// findInputAssets 查找输入资产
func (e *StateMachineEngine) findInputAssets(asset *domain.SoftwareAsset, inputAssetTypes []*domain.AssetType) []*domain.ConcreteAsset {
	if len(inputAssetTypes) == 0 {
		return nil
	}

	inputAssets := make([]*domain.ConcreteAsset, 0)
	currentStateAssets := asset.GetConcreteAssetsByState(asset.CurrentState)

	for _, assetType := range inputAssetTypes {
		for _, asset := range currentStateAssets {
			if asset.AssetType.ID == assetType.ID {
				inputAssets = append(inputAssets, asset)
				break
			}
		}
	}

	return inputAssets
}

// createOutputAssets 创建输出资产
func (e *StateMachineEngine) createOutputAssets(
	asset *domain.SoftwareAsset,
	assetTypes []*domain.AssetType,
	state *domain.State,
	transition *domain.StateTransition,
	inputAssets []*domain.ConcreteAsset,
	execResult *ExecutionResult,
) {
	for _, assetType := range assetTypes {
		// 从执行结果中获取资产位置
		location := execResult.AssetLocations[assetType.ID]
		if location == "" {
			// 如果没有指定位置，使用默认路径
			location = e.generateDefaultAssetLocation(assetType, e.assetManager.GetWorkspacePath())
		}

		outputAsset := domain.NewConcreteAsset(
			domain.GenerateID(),
			assetType.Name+"-"+asset.Name,
			assetType,
			location,
			"", // version
		)
		outputAsset.SetGeneratedContext(state, transition)

		// 设置输入资产依赖
		for _, inputAsset := range inputAssets {
			outputAsset.AddInputAsset(inputAsset)
		}

		// 添加执行结果元数据
		outputAsset.AddMetadata("executionResultId", transition.ActionResult.ID)
		outputAsset.AddMetadata("actionId", transition.Action.ID)

		asset.AddConcreteAsset(outputAsset)
	}
}

// generateDefaultAssetLocation 生成默认资产位置
func (e *StateMachineEngine) generateDefaultAssetLocation(assetType *domain.AssetType, workspacePath string) string {
	// 根据资产类型生成默认路径
	switch assetType.ID {
	case "source-code":
		return domain.JoinPath(workspacePath, "source")
	case "jar-file":
		return domain.JoinPath(workspacePath, "build")
	case "container":
		return "container-" + domain.GenerateID()[:8]
	default:
		return domain.JoinPath(workspacePath, "assets", assetType.ID)
	}
}
