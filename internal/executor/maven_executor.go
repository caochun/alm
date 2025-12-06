package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alm/internal/engine"
)

// MavenExecutor Maven执行器
type MavenExecutor struct{}

// NewMavenExecutor 创建Maven执行器
func NewMavenExecutor() engine.Executor {
	return &MavenExecutor{}
}

// Execute 执行maven build
func (e *MavenExecutor) Execute(ctx *engine.ExecutionContext) (*engine.ExecutionResult, error) {
	result := &engine.ExecutionResult{
		Metadata:       make(map[string]interface{}),
		AssetLocations: make(map[string]string),
	}

	// 从输入资产中获取源代码目录
	var sourceDir string
	if len(ctx.InputAssets) > 0 {
		sourceDir = ctx.InputAssets[0].Location
	} else {
		// 如果没有输入资产，尝试从workspace中查找
		sourceDir = filepath.Join(ctx.WorkspacePath, "source")
		// 查找第一个子目录
		entries, err := os.ReadDir(sourceDir)
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if entry.IsDir() {
					sourceDir = filepath.Join(sourceDir, entry.Name())
					break
				}
			}
		}
	}

	if sourceDir == "" {
		result.Success = false
		result.ErrorMessage = "source directory not found"
		return result, nil
	}

	// 检查pom.xml是否存在
	pomPath := filepath.Join(sourceDir, "pom.xml")
	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("pom.xml not found in %s", sourceDir)
		return result, nil
	}

	// 执行maven build
	cmd := exec.Command("mvn", "clean", "package", "-DskipTests")
	cmd.Dir = sourceDir
	output, err := cmd.CombinedOutput()

	result.Output = string(output)
	result.Metadata["sourceDir"] = sourceDir
	result.Metadata["command"] = "mvn clean package -DskipTests"

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, nil
	}

	// 查找生成的jar文件
	jarFile := e.findJarFile(sourceDir)
	if jarFile != "" {
		result.AssetLocations["jar-file"] = jarFile
		result.Metadata["jarFile"] = jarFile
	} else {
		// 即使找不到jar文件，也认为构建成功（可能是其他类型的构建产物）
		result.Metadata["jarFile"] = "not found"
	}

	result.Success = true
	return result, nil
}

// findJarFile 查找生成的jar文件
func (e *MavenExecutor) findJarFile(sourceDir string) string {
	targetDir := filepath.Join(sourceDir, "target")
	
	// 查找target目录下的jar文件
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jar") && !strings.HasSuffix(entry.Name(), "-sources.jar") {
			return filepath.Join(targetDir, entry.Name())
		}
	}

	return ""
}

