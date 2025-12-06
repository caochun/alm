package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alm/domain"
)

const assetStateFile = ".alm-state.json"

// SaveAssetState 保存资产状态到文件（公开方法）
func (am *AssetManager) SaveAssetState() error {
	return am.saveAssetState()
}

// saveAssetState 保存资产状态到文件
func (am *AssetManager) saveAssetState() error {
	if am.asset == nil {
		return nil
	}

	stateData := AssetStateData{
		CurrentStateID:    am.asset.CurrentState.ID,
		ConcreteAssets:    make([]ConcreteAssetData, 0),
		TransitionHistory: make([]TransitionHistoryData, 0),
	}

	// 保存concrete assets
	for _, ca := range am.asset.ConcreteAssets {
		stateData.ConcreteAssets = append(stateData.ConcreteAssets, ConcreteAssetData{
			ID:               ca.ID,
			Name:             ca.Name,
			AssetTypeID:      ca.AssetType.ID,
			Location:         ca.Location,
			Version:          ca.Version,
			GeneratedAtState: ca.GeneratedAtState.ID,
			Metadata:         ca.Metadata,
			InputAssetIDs:    getInputAssetIDs(ca.InputAssets),
		})
	}

	// 保存转换历史（只保存最近的，避免文件过大）
	for _, trans := range am.asset.TransitionHistory {
		historyItem := TransitionHistoryData{
			ID:        trans.ID,
			FromState: trans.FromState.ID,
			ToState:   trans.ToState.ID,
			ActionID:  trans.Action.ID,
			Operator:  trans.Operator,
			Timestamp: trans.Timestamp.Format(time.RFC3339),
		}
		if trans.ActionResult != nil {
			historyItem.Result = &ActionResultData{
				Success:      trans.ActionResult.IsSuccess(),
				Status:       string(trans.ActionResult.Status),
				Output:       trans.ActionResult.Output,
				ErrorMessage: trans.ActionResult.ErrorMessage,
			}
		}
		stateData.TransitionHistory = append(stateData.TransitionHistory, historyItem)
	}

	// 保存到文件
	statePath := filepath.Join(am.workspacePath, assetStateFile)
	data, err := json.MarshalIndent(stateData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal asset state: %w", err)
	}

	return os.WriteFile(statePath, data, 0644)
}

// loadAssetState 从文件加载资产状态
func (am *AssetManager) loadAssetState(asset *domain.SoftwareAsset) error {
	statePath := filepath.Join(am.workspacePath, assetStateFile)

	// 如果文件不存在，返回nil（使用默认初始状态）
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read asset state: %w", err)
	}

	var stateData AssetStateData
	if err := json.Unmarshal(data, &stateData); err != nil {
		return fmt.Errorf("failed to unmarshal asset state: %w", err)
	}

	// 恢复当前状态
	if stateData.CurrentStateID != "" {
		for _, state := range am.template.States {
			if state.ID == stateData.CurrentStateID {
				asset.CurrentState = state
				break
			}
		}
	}

	// 恢复concrete assets
	for _, caData := range stateData.ConcreteAssets {
		assetType, ok := am.assetTypes[caData.AssetTypeID]
		if !ok {
			continue
		}

		ca := domain.NewConcreteAsset(
			caData.ID,
			caData.Name,
			assetType,
			caData.Location,
			caData.Version,
		)

		// 恢复生成状态
		for _, state := range am.template.States {
			if state.ID == caData.GeneratedAtState {
				ca.SetGeneratedContext(state, nil)
				break
			}
		}

		// 恢复metadata
		for k, v := range caData.Metadata {
			ca.AddMetadata(k, v)
		}

		asset.AddConcreteAsset(ca)
	}

	// 恢复输入资产依赖关系
	for _, caData := range stateData.ConcreteAssets {
		ca := asset.FindConcreteAssetByID(caData.ID)
		if ca == nil {
			continue
		}

		for _, inputID := range caData.InputAssetIDs {
			inputAsset := asset.FindConcreteAssetByID(inputID)
			if inputAsset != nil {
				ca.AddInputAsset(inputAsset)
			}
		}
	}

	// 恢复转换历史（简化版，只恢复基本信息）
	for _, histData := range stateData.TransitionHistory {
		fromState := am.findStateByID(histData.FromState)
		toState := am.findStateByID(histData.ToState)
		action := am.actions[histData.ActionID]

		if fromState == nil || toState == nil || action == nil {
			continue
		}

		// 解析时间戳
		timestamp, err := time.Parse(time.RFC3339, histData.Timestamp)
		if err != nil {
			// 如果解析失败，使用当前时间
			timestamp = time.Now()
		}

		trans := &domain.StateTransition{
			ID:            histData.ID,
			SoftwareAsset: asset,
			FromState:     fromState,
			ToState:       toState,
			Action:        action,
			Timestamp:     timestamp,
			Operator:      histData.Operator,
		}

		if histData.Result != nil {
			var status domain.ExecutionStatus
			if histData.Result.Success {
				status = domain.ExecutionStatusSuccess
			} else {
				status = domain.ExecutionStatusFailed
			}

			trans.ActionResult = &domain.ActionExecutionResult{
				ID:           domain.GenerateID(),
				Action:       action,
				Status:       status,
				Output:       histData.Result.Output,
				ErrorMessage: histData.Result.ErrorMessage,
			}
		}

		asset.TransitionHistory = append(asset.TransitionHistory, trans)
	}

	return nil
}

// findStateByID 根据ID查找状态
func (am *AssetManager) findStateByID(stateID string) *domain.State {
	for _, state := range am.template.States {
		if state.ID == stateID {
			return state
		}
	}
	return nil
}

// getInputAssetIDs 获取输入资产ID列表
func getInputAssetIDs(inputAssets []*domain.ConcreteAsset) []string {
	ids := make([]string, 0, len(inputAssets))
	for _, asset := range inputAssets {
		ids = append(ids, asset.ID)
	}
	return ids
}

// AssetStateData 资产状态数据（用于持久化）
type AssetStateData struct {
	CurrentStateID    string                  `json:"current_state_id"`
	ConcreteAssets    []ConcreteAssetData     `json:"concrete_assets"`
	TransitionHistory []TransitionHistoryData `json:"transition_history"`
}

// ConcreteAssetData 具体资产数据
type ConcreteAssetData struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	AssetTypeID      string                 `json:"asset_type_id"`
	Location         string                 `json:"location"`
	Version          string                 `json:"version"`
	GeneratedAtState string                 `json:"generated_at_state"`
	Metadata         map[string]interface{} `json:"metadata"`
	InputAssetIDs    []string               `json:"input_asset_ids"`
}

// TransitionHistoryData 转换历史数据
type TransitionHistoryData struct {
	ID        string            `json:"id"`
	FromState string            `json:"from_state"`
	ToState   string            `json:"to_state"`
	ActionID  string            `json:"action_id"`
	Operator  string            `json:"operator"`
	Timestamp string            `json:"timestamp"`
	Result    *ActionResultData `json:"result,omitempty"`
}

// ActionResultData 动作执行结果数据
type ActionResultData struct {
	Success      bool   `json:"success"`
	Status       string `json:"status"`
	Output       string `json:"output"`
	ErrorMessage string `json:"error_message"`
}
