package domain

import "time"

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusSuccess    ExecutionStatus = "SUCCESS"
	ExecutionStatusFailed     ExecutionStatus = "FAILED"
	ExecutionStatusInProgress ExecutionStatus = "IN_PROGRESS"
	ExecutionStatusCancelled  ExecutionStatus = "CANCELLED"
)

// ActionExecutionResult 动作执行结果
type ActionExecutionResult struct {
	ID           string
	Action       *Action
	Status       ExecutionStatus
	Output       string
	ErrorMessage string
	Metadata     map[string]interface{}
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
}

// NewActionExecutionResult 创建新的执行结果
func NewActionExecutionResult(id string, action *Action) *ActionExecutionResult {
	return &ActionExecutionResult{
		ID:        id,
		Action:    action,
		Status:    ExecutionStatusInProgress,
		Metadata:  make(map[string]interface{}),
		StartTime: time.Now(),
	}
}

// IsSuccess 检查是否成功
func (aer *ActionExecutionResult) IsSuccess() bool {
	return aer.Status == ExecutionStatusSuccess
}

// IsFailed 检查是否失败
func (aer *ActionExecutionResult) IsFailed() bool {
	return aer.Status == ExecutionStatusFailed
}

// Complete 完成执行
func (aer *ActionExecutionResult) Complete(status ExecutionStatus, output string, err error) {
	aer.EndTime = time.Now()
	aer.Duration = aer.EndTime.Sub(aer.StartTime)
	aer.Status = status
	aer.Output = output
	if err != nil {
		aer.ErrorMessage = err.Error()
	}
}

