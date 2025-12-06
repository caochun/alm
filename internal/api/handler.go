package api

import (
	"fmt"
	"net/http"

	"github.com/alm/domain"
	"github.com/alm/internal/engine"
	"github.com/alm/internal/executor"
	"github.com/alm/internal/manager"
	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	managerFactory *manager.AssetManagerFactory
}

// NewHandler 创建API处理器
func NewHandler(managerFactory *manager.AssetManagerFactory) *Handler {
	return &Handler{
		managerFactory: managerFactory,
	}
}

// getAssetManager 获取资产管理器（根据appPath）
func (h *Handler) getAssetManager(c *gin.Context) (*manager.AssetManager, error) {
	appPath := c.Query("appPath")
	if appPath == "" {
		return nil, fmt.Errorf("appPath parameter is required")
	}

	return h.managerFactory.GetManager(appPath)
}

// GetAsset 获取资产信息
func (h *Handler) GetAsset(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset := assetManager.GetAsset()
	config := assetManager.GetConfig().(*manager.AssetConfig)

	c.JSON(http.StatusOK, gin.H{
		"id":          asset.ID,
		"name":        asset.Name,
		"description": asset.Description,
		"currentState": gin.H{
			"id":   asset.CurrentState.ID,
			"name": asset.CurrentState.Name,
		},
		"workspace": assetManager.GetWorkspacePath(),
		"config":    config,
		"createdAt": asset.CreatedAt,
		"updatedAt": asset.UpdatedAt,
	})
}

// GetStates 获取所有状态
func (h *Handler) GetStates(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := assetManager.GetTemplate()
	states := make([]gin.H, 0)

	for _, state := range template.States {
		// 获取该状态的可用actions
		availableActions := make([]gin.H, 0)
		for _, action := range state.AvailableActions {
			availableActions = append(availableActions, gin.H{
				"id":          action.ID,
				"name":        action.Name,
				"type":        string(action.Type),
				"description": action.Description,
			})
		}

		states = append(states, gin.H{
			"id":               state.ID,
			"name":             state.Name,
			"description":      state.Description,
			"availableActions": availableActions,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"states": states,
	})
}

// GetCurrentState 获取当前状态
func (h *Handler) GetCurrentState(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset := assetManager.GetAsset()
	template := assetManager.GetTemplate()

	// 获取可用转换
	availableTransitions := template.GetAvailableTransitions(asset.CurrentState)
	transitions := make([]gin.H, 0)

	for _, trans := range availableTransitions {
		transitions = append(transitions, gin.H{
			"toState": gin.H{
				"id":   trans.ToState.ID,
				"name": trans.ToState.Name,
			},
			"action": gin.H{
				"id":          trans.RequiredAction.ID,
				"name":        trans.RequiredAction.Name,
				"description": trans.RequiredAction.Description,
				"type":        string(trans.RequiredAction.Type),
			},
		})
	}

	// 获取当前状态下的concrete assets
	currentStateAssets := asset.GetConcreteAssetsByState(asset.CurrentState)
	assets := make([]gin.H, 0)
	for _, ca := range currentStateAssets {
		assets = append(assets, gin.H{
			"id":       ca.ID,
			"name":     ca.Name,
			"type":     ca.AssetType.Name,
			"location": ca.Location,
			"version":  ca.Version,
		})
	}

	// 获取当前状态可用的actions
	availableActions := make([]gin.H, 0)
	if asset.CurrentState.AvailableActions != nil {
		for _, action := range asset.CurrentState.AvailableActions {
			availableActions = append(availableActions, gin.H{
				"id":          action.ID,
				"name":        action.Name,
				"type":        string(action.Type),
				"description": action.Description,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"currentState": gin.H{
			"id":   asset.CurrentState.ID,
			"name": asset.CurrentState.Name,
		},
		"availableTransitions": transitions,
		"assets":               assets,
		"availableActions":     availableActions,
	})
}

// GetAssets 获取所有具体资产
func (h *Handler) GetAssets(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset := assetManager.GetAsset()
	assets := make([]gin.H, 0)

	allAssets := asset.GetAllConcreteAssets()
	for _, ca := range allAssets {
		inputAssets := make([]string, 0)
		for _, input := range ca.InputAssets {
			inputAssets = append(inputAssets, input.ID)
		}

		assets = append(assets, gin.H{
			"id":        ca.ID,
			"name":      ca.Name,
			"type":      ca.AssetType.Name,
			"location":  ca.Location,
			"version":   ca.Version,
			"state":     ca.GeneratedAtState.Name,
			"inputs":    inputAssets,
			"createdAt": ca.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"assets": assets,
	})
}

// getStateName 安全获取状态名称
func getStateName(state *domain.State) string {
	if state == nil {
		return ""
	}
	return state.Name
}

// GetTransitionHistory 获取转换历史
func (h *Handler) GetTransitionHistory(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset := assetManager.GetAsset()
	history := make([]gin.H, 0)

	for _, trans := range asset.TransitionHistory {
		historyItem := gin.H{
			"id":        trans.ID,
			"fromState": trans.FromState.Name,
			"toState":   trans.ToState.Name,
			"action":    trans.Action.Name,
			"operator":  trans.Operator,
			"timestamp": trans.Timestamp,
		}

		if trans.ActionResult != nil {
			historyItem["result"] = gin.H{
				"success": trans.ActionResult.IsSuccess(),
				"status":  string(trans.ActionResult.Status),
				"error":   trans.ActionResult.ErrorMessage,
			}
		}

		history = append(history, historyItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// TransitionRequest 状态转换请求
type TransitionRequest struct {
	ToState    string                 `json:"toState" binding:"required"`
	Conditions map[string]interface{} `json:"conditions"`
	Operator   string                 `json:"operator"`
}

// Transition 执行状态转换
func (h *Handler) Transition(c *gin.Context) {
	var req TransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Operator == "" {
		req.Operator = "system"
	}

	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	executorFactory := executor.NewDefaultExecutorFactory()
	stateEngine := engine.NewStateMachineEngine(assetManager, executorFactory)

	transition, err := stateEngine.TransitionTo(req.ToState, req.Conditions, req.Operator)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := gin.H{
		"id":        transition.ID,
		"fromState": transition.FromState.Name,
		"toState":   transition.ToState.Name,
		"operator":  transition.Operator,
		"timestamp": transition.Timestamp,
	}

	if transition.ActionResult != nil {
		result["result"] = gin.H{
			"success": transition.ActionResult.IsSuccess(),
			"status":  string(transition.ActionResult.Status),
			"output":  transition.ActionResult.Output,
			"error":   transition.ActionResult.ErrorMessage,
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetStateMachineGraph 获取状态机graph数据（用于可视化）
func (h *Handler) GetStateMachineGraph(c *gin.Context) {
	assetManager, err := h.getAssetManager(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := assetManager.GetTemplate()
	asset := assetManager.GetAsset()

	// 构建节点（状态）
	nodes := make([]gin.H, 0)
	for _, state := range template.States {
		node := gin.H{
			"id":    state.ID,
			"label": state.Name,
			"type":  "state",
		}

		// 标记当前状态
		if asset.CurrentState != nil && state.ID == asset.CurrentState.ID {
			node["current"] = true
			node["color"] = "#4CAF50" // 绿色表示当前状态
		} else {
			node["color"] = "#E0E0E0" // 灰色表示其他状态
		}

		nodes = append(nodes, node)
	}

	// 构建边（转换规则）
	edges := make([]gin.H, 0)
	for _, rule := range template.TransitionRules {
		edge := gin.H{
			"from":  rule.FromState.ID,
			"to":    rule.ToState.ID,
			"label": "",
		}

		if rule.RequiredAction != nil {
			edge["label"] = rule.RequiredAction.Name
		}

		edges = append(edges, edge)
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"edges": edges,
	})
}
