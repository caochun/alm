package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alm/domain"
	"github.com/alm/dsl"
	"gopkg.in/yaml.v3"
)

// AssetConfig 资产配置
type AssetConfig struct {
	ID                   string                 `yaml:"id"`
	Name                 string                 `yaml:"name"`
	Description          string                 `yaml:"description"`
	StateMachineTemplate string                 `yaml:"state_machine_template"`
	Workspace            WorkspaceConfig        `yaml:"workspace"`
	Application          map[string]interface{} `yaml:"application"`
	Metadata             map[string]interface{} `yaml:"metadata"`
}

// WorkspaceConfig 工作空间配置
type WorkspaceConfig struct {
	SourceDir string `yaml:"source_dir"`
	BuildDir  string `yaml:"build_dir"`
	DeployDir string `yaml:"deploy_dir"`
	AssetsDir string `yaml:"assets_dir"`
}

// AssetManager 资产管理器
type AssetManager struct {
	workspacePath string
	assetConfig   *AssetConfig
	asset         *domain.SoftwareAsset
	template      *domain.StateMachineTemplate
	assetTypes    map[string]*domain.AssetType
	actions       map[string]*domain.Action
}

// NewAssetManager 创建资产管理器
func NewAssetManager(workspacePath string) (*AssetManager, error) {
	am := &AssetManager{
		workspacePath: workspacePath,
	}

	// 加载asset.yaml配置
	if err := am.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 加载状态机模板
	if err := am.loadStateMachineTemplate(); err != nil {
		return nil, fmt.Errorf("failed to load state machine template: %w", err)
	}

	// 初始化工作目录
	if err := am.initWorkspace(); err != nil {
		return nil, fmt.Errorf("failed to init workspace: %w", err)
	}

	// 创建或加载软件资产
	if err := am.loadOrCreateAsset(); err != nil {
		return nil, fmt.Errorf("failed to load or create asset: %w", err)
	}

	return am, nil
}

// loadConfig 加载asset.yaml配置
func (am *AssetManager) loadConfig() error {
	configPath := filepath.Join(am.workspacePath, "asset.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config AssetConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	am.assetConfig = &config
	return nil
}

// loadStateMachineTemplate 加载状态机模板
func (am *AssetManager) loadStateMachineTemplate() error {
	// 解析模板路径（相对于workspace目录）
	templatePath := am.assetConfig.StateMachineTemplate
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(am.workspacePath, templatePath)
	}

	template, assetTypes, actions, err := dsl.ParseStateMachine(templatePath)
	if err != nil {
		return err
	}

	am.template = template
	am.assetTypes = assetTypes
	am.actions = actions
	return nil
}

// initWorkspace 初始化工作目录
func (am *AssetManager) initWorkspace() error {
	dirs := []string{
		am.assetConfig.Workspace.SourceDir,
		am.assetConfig.Workspace.BuildDir,
		am.assetConfig.Workspace.DeployDir,
		am.assetConfig.Workspace.AssetsDir,
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		fullPath := filepath.Join(am.workspacePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

// loadOrCreateAsset 加载或创建软件资产
func (am *AssetManager) loadOrCreateAsset() error {
	// 创建新的资产（使用初始状态）
	asset, err := domain.NewSoftwareAsset(
		am.assetConfig.ID,
		am.assetConfig.Name,
		am.assetConfig.Description,
		am.template,
	)
	if err != nil {
		return err
	}

	// 尝试从持久化存储加载资产状态
	if err := am.loadAssetState(asset); err != nil {
		// 如果加载失败，使用默认初始状态（不返回错误，允许继续使用新创建的资产）
		// 这允许首次运行时创建新资产
	}

	am.asset = asset
	return nil
}

// GetAsset 获取软件资产
func (am *AssetManager) GetAsset() *domain.SoftwareAsset {
	return am.asset
}

// GetWorkspacePath 获取工作空间路径
func (am *AssetManager) GetWorkspacePath() string {
	return am.workspacePath
}

// GetConfig 获取配置（实现AssetManagerInterface接口）
func (am *AssetManager) GetConfig() interface{} {
	return am.assetConfig
}

// GetTemplate 获取状态机模板
func (am *AssetManager) GetTemplate() *domain.StateMachineTemplate {
	return am.template
}

// GetActions 获取动作映射
func (am *AssetManager) GetActions() map[string]*domain.Action {
	return am.actions
}
