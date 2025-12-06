package executor

import (
	"fmt"

	"github.com/alm/internal/engine"
)

// DefaultExecutorFactory 默认执行器工厂
type DefaultExecutorFactory struct {
	executors map[string]func() engine.Executor
}

// NewDefaultExecutorFactory 创建默认执行器工厂
func NewDefaultExecutorFactory() *DefaultExecutorFactory {
	factory := &DefaultExecutorFactory{
		executors: make(map[string]func() engine.Executor),
	}

	// 注册执行器
	factory.RegisterExecutor("git-clone", NewGitExecutor)
	factory.RegisterExecutor("maven-build", NewMavenExecutor)
	factory.RegisterExecutor("terraform-deploy", NewTerraformExecutor)

	return factory
}

// RegisterExecutor 注册执行器
func (f *DefaultExecutorFactory) RegisterExecutor(actionID string, creator func() engine.Executor) {
	f.executors[actionID] = creator
}

// CreateExecutor 创建执行器
func (f *DefaultExecutorFactory) CreateExecutor(actionID string) (engine.Executor, error) {
	creator, ok := f.executors[actionID]
	if !ok {
		return nil, fmt.Errorf("executor not found for action: %s", actionID)
	}
	return creator(), nil
}

