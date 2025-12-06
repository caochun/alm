package executor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alm/internal/engine"
)

// TerraformExecutor Terraform执行器
type TerraformExecutor struct{}

// NewTerraformExecutor 创建Terraform执行器
func NewTerraformExecutor() engine.Executor {
	return &TerraformExecutor{}
}

// Execute 执行terraform deploy
func (e *TerraformExecutor) Execute(ctx *engine.ExecutionContext) (*engine.ExecutionResult, error) {
	result := &engine.ExecutionResult{
		Metadata:       make(map[string]interface{}),
		AssetLocations: make(map[string]string),
	}

	// 从输入资产中获取jar文件路径
	var jarPath string
	if len(ctx.InputAssets) > 0 {
		jarPath = ctx.InputAssets[0].Location
	}

	if jarPath == "" {
		result.Success = false
		result.ErrorMessage = "jar file path not found"
		return result, nil
	}

	// 确定terraform工作目录
	deployDir := filepath.Join(ctx.WorkspacePath, "deploy")
	
	// 如果deploy目录不存在，创建它
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("failed to create deploy directory: %v", err)
		return result, nil
	}

	// 检查是否有terraform配置文件
	// 如果没有，创建一个简单的配置（实际应用中应该由用户提供）
	tfFile := filepath.Join(deployDir, "main.tf")
	if _, err := os.Stat(tfFile); os.IsNotExist(err) {
		// 创建简单的terraform配置
		if err := e.createTerraformConfig(tfFile, jarPath, ctx); err != nil {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("failed to create terraform config: %v", err)
			return result, nil
		}
	}

	// 执行terraform init
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = deployDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("terraform init failed: %s", string(initOutput))
		return result, nil
	}

	// 执行terraform apply
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = deployDir
	applyOutput, err := applyCmd.CombinedOutput()

	result.Output = fmt.Sprintf("Init output:\n%s\n\nApply output:\n%s", string(initOutput), string(applyOutput))
	result.Metadata["deployDir"] = deployDir
	result.Metadata["jarPath"] = jarPath
	result.Metadata["command"] = "terraform apply -auto-approve"

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, nil
	}

	// 提取容器ID（简化处理，实际应该从terraform output中获取）
	containerID := fmt.Sprintf("container-%s", ctx.Asset.ID[:8])
	result.AssetLocations["container"] = containerID
	result.Metadata["containerId"] = containerID

	result.Success = true
	return result, nil
}

// findAvailablePort 查找一个可用的端口
func (e *TerraformExecutor) findAvailablePort(startPort, endPort int) int {
	for port := startPort; port <= endPort; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port
		}
	}
	// 如果找不到可用端口，返回默认值
	return 8080
}

// createTerraformConfig 创建terraform配置文件
func (e *TerraformExecutor) createTerraformConfig(tfFile, jarPath string, ctx *engine.ExecutionContext) error {
	// 创建一个简单的Docker部署配置
	// 使用绝对路径
	absJarPath, err := filepath.Abs(jarPath)
	if err != nil {
		absJarPath = jarPath
	}
	
	// 查找一个可用的端口（8000-9000范围）
	availablePort := e.findAvailablePort(8000, 9000)
	
	config := fmt.Sprintf(`# Terraform configuration for %s
# This is a simplified example. In production, you should provide your own terraform configuration.

terraform {
  required_version = ">= 1.0"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

# Docker container resource
resource "docker_container" "app" {
  name  = "%s"
  image = "eclipse-temurin:17-jre"
  
  command = ["java", "-jar", "/app/app.jar"]
  
  volumes {
    host_path      = "%s"
    container_path = "/app/app.jar"
  }
  
  ports {
    internal = 8080
    external = %d
  }
}

output "container_id" {
  value = docker_container.app.id
}

output "external_port" {
  value = %d
}
`, ctx.Asset.Name, ctx.Asset.ID, absJarPath, availablePort, availablePort)

	return os.WriteFile(tfFile, []byte(config), 0644)
}

