package domain

import (
	"context"
	"os/exec"
	"time"
)

// ActionType 动作类型
type ActionType string

const (
	ActionTypeTool   ActionType = "TOOL_ACTION"   // 工具动作
	ActionTypeManual ActionType = "MANUAL_ACTION" // 人为动作
)

// Action 动作
type Action struct {
	ID          string
	Name        string
	Type        ActionType
	Description string
	Parameters  map[string]interface{}
	Command     string   // 工具动作的命令
	Args        []string // 工具动作的参数
}

// NewAction 创建新动作
func NewAction(id, name string, actionType ActionType, description string) *Action {
	return &Action{
		ID:          id,
		Name:        name,
		Type:        actionType,
		Description: description,
		Parameters:  make(map[string]interface{}),
		Args:        make([]string, 0),
	}
}

// SetCommand 设置工具命令
func (a *Action) SetCommand(command string, args []string) {
	a.Command = command
	a.Args = args
}

// AddParameter 添加参数
func (a *Action) AddParameter(key string, value interface{}) {
	a.Parameters[key] = value
}

// Execute 执行动作
func (a *Action) Execute(context map[string]interface{}) *ActionExecutionResult {
	result := &ActionExecutionResult{
		ID:        GenerateID(),
		Action:    a,
		Status:    ExecutionStatusInProgress,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// 如果是工具动作，执行命令
	if a.Type == ActionTypeTool {
		if a.Command == "" {
			result.Status = ExecutionStatusFailed
			result.ErrorMessage = "command is empty"
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		// 构建命令参数（可以替换参数中的变量）
		args := a.buildArgs(context)

		// 执行命令
		cmd := exec.Command(a.Command, args...)
		output, err := cmd.CombinedOutput()

		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if err != nil {
			result.Status = ExecutionStatusFailed
			result.ErrorMessage = err.Error()
			result.Output = string(output)
		} else {
			result.Status = ExecutionStatusSuccess
			result.Output = string(output)
		}

		// 添加执行元数据
		result.Metadata["command"] = a.Command
		result.Metadata["args"] = args
		result.Metadata["exitCode"] = cmd.ProcessState.ExitCode()
	} else {
		// 人为动作，只记录，不执行
		result.Status = ExecutionStatusSuccess
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Output = "Manual action, waiting for operator"
	}

	return result
}

// buildArgs 构建命令参数（可以替换变量）
func (a *Action) buildArgs(context map[string]interface{}) []string {
	args := make([]string, 0)

	for _, arg := range a.Args {
		// 简单的变量替换：${variableName}
		replaced := a.replaceVariables(arg, context)
		args = append(args, replaced)
	}

	return args
}

// replaceVariables 替换参数中的变量
func (a *Action) replaceVariables(template string, context map[string]interface{}) string {
	result := template

	// 替换 ${variableName} 格式的变量
	// 首先从context中查找
	if inputAssets, ok := context["inputAssets"].([]*ConcreteAsset); ok && len(inputAssets) > 0 {
		// 特殊处理：如果变量是sourceDir或jarPath，从输入资产中提取
		if template == "${sourceDir}" || template == "${outputDir}" {
			if len(inputAssets) > 0 {
				return inputAssets[0].Location
			}
		}
		if template == "${jarPath}" {
			for _, asset := range inputAssets {
				if asset.AssetType != nil && asset.AssetType.ID == "jar-file" {
					return asset.Location
				}
			}
		}
	}

	// 从context中查找其他变量
	for key, value := range context {
		placeholder := "${" + key + "}"
		if result == placeholder {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}

	// 从Parameters中查找
	for key, value := range a.Parameters {
		placeholder := "${" + key + "}"
		if result == placeholder {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}

	return result
}

// ExecuteAsync 异步执行动作
func (a *Action) ExecuteAsync(ctx context.Context, context map[string]interface{}) <-chan *ActionExecutionResult {
	resultChan := make(chan *ActionExecutionResult, 1)
	go func() {
		result := a.Execute(context)
		resultChan <- result
		close(resultChan)
	}()
	return resultChan
}
