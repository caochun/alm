package manager

import (
	"fmt"
	"path/filepath"
	"sync"
)

// AssetManagerFactory 资产管理器工厂
type AssetManagerFactory struct {
	workspaceRoot string
	managers      map[string]*AssetManager
	mu            sync.RWMutex
}

// NewAssetManagerFactory 创建资产管理器工厂
func NewAssetManagerFactory(workspaceRoot string) *AssetManagerFactory {
	return &AssetManagerFactory{
		workspaceRoot: workspaceRoot,
		managers:      make(map[string]*AssetManager),
	}
}

// GetManager 获取或创建资产管理器
func (f *AssetManagerFactory) GetManager(appPath string) (*AssetManager, error) {
	f.mu.RLock()
	manager, exists := f.managers[appPath]
	f.mu.RUnlock()

	if exists {
		return manager, nil
	}

	// 创建新的管理器
	f.mu.Lock()
	defer f.mu.Unlock()

	// 双重检查
	if manager, exists := f.managers[appPath]; exists {
		return manager, nil
	}

	// 构建完整路径
	fullPath := filepath.Join(f.workspaceRoot, appPath)

	manager, err := NewAssetManager(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager for %s: %w", appPath, err)
	}

	f.managers[appPath] = manager
	return manager, nil
}

// GetWorkspaceRoot 获取工作空间根目录
func (f *AssetManagerFactory) GetWorkspaceRoot() string {
	return f.workspaceRoot
}

