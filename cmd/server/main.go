package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/alm/internal/api"
)

func main() {
	workspaceRoot := flag.String("workspace", "", "工作空间根目录（包含多个应用目录）")
	webRoot := flag.String("web", "", "Web UI 目录（包含 dist/ 或 index.html，留空则不提供 Web UI）")
	pipelinesDir := flag.String("pipelines", "", "Pipeline 模板目录（默认为 dsl/templates）")
	port := flag.String("port", "8080", "服务器端口")
	flag.Parse()

	if *workspaceRoot == "" {
		log.Fatal("请指定工作空间根目录: -workspace <path>")
	}

	// Default pipelines dir relative to binary location
	if *pipelinesDir == "" {
		*pipelinesDir = filepath.Join("dsl", "templates")
	}

	handler := api.NewHandler(*workspaceRoot, *pipelinesDir)
	router := api.SetupRouter(handler, *webRoot)

	addr := fmt.Sprintf(":%s", *port)
	fmt.Printf("ALM Server 启动于 http://localhost%s\n", addr)
	fmt.Printf("  workspace: %s\n", *workspaceRoot)
	fmt.Printf("  pipelines: %s\n", *pipelinesDir)
	if *webRoot != "" {
		fmt.Printf("  web:       %s\n", *webRoot)
	}
	fmt.Println()
	fmt.Printf("  API: GET  http://localhost%s/api/v1/apps\n", addr)
	fmt.Printf("       GET  http://localhost%s/api/v1/apps/:app/envs\n", addr)
	fmt.Printf("       GET  http://localhost%s/api/v1/graph?app=xxx&env=yyy\n", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
