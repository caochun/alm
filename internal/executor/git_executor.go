package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alm/internal/engine"
)

// GitExecutor Git执行器
type GitExecutor struct{}

// NewGitExecutor 创建Git执行器
func NewGitExecutor() engine.Executor {
	return &GitExecutor{}
}

// Execute 执行git clone
func (e *GitExecutor) Execute(ctx *engine.ExecutionContext) (*engine.ExecutionResult, error) {
	result := &engine.ExecutionResult{
		Metadata:       make(map[string]interface{}),
		AssetLocations: make(map[string]string),
	}

	// 从conditions中获取repository
	repository, ok := ctx.Conditions["repository"].(string)
	if !ok {
		result.Success = false
		result.ErrorMessage = "repository not found in conditions"
		return result, nil
	}

	// 确定克隆目标目录
	targetDir := filepath.Join(ctx.WorkspacePath, "source")
	
	// 从repository URL提取目录名
	repoName := extractRepoName(repository)
	if repoName != "" {
		targetDir = filepath.Join(ctx.WorkspacePath, "source", repoName)
	}

	// 如果目录已存在，跳过clone
	if _, err := os.Stat(targetDir); err == nil {
		result.Success = true
		result.Output = fmt.Sprintf("Directory already exists: %s", targetDir)
		result.AssetLocations["source-code"] = targetDir
		result.Metadata["repository"] = repository
		result.Metadata["targetDir"] = targetDir
		return result, nil
	}

	// 执行git clone
	cmd := exec.Command("git", "clone", repository, targetDir)
	output, err := cmd.CombinedOutput()

	result.Output = string(output)
	result.Metadata["repository"] = repository
	result.Metadata["targetDir"] = targetDir
	result.Metadata["command"] = fmt.Sprintf("git clone %s %s", repository, targetDir)

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, nil
	}

	result.Success = true
	result.AssetLocations["source-code"] = targetDir
	return result, nil
}

// extractRepoName 从repository URL提取仓库名
func extractRepoName(repoURL string) string {
	// 移除.git后缀
	repoURL = strings.TrimSuffix(repoURL, ".git")
	
	// 提取最后一部分
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

