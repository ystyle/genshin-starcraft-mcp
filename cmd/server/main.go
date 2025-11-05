package main

import (
	"os"

	"genshin-starcraft-mcp/pkg/mcp"
	"genshin-starcraft-mcp/pkg/utils"
)

func main() {
	// 初始化日志系统
	if err := utils.InitLogger(); err != nil {
		os.Exit(1)
	}

	utils.Info("Starting Genshin Starcraft MCP Server...")

	// 创建MCP服务器
	server, err := mcp.NewGenshinStarcraftMCPServer()
	if err != nil {
		utils.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}
	defer server.Close()

	// 启动服务器
	if err := server.Start(); err != nil {
		utils.Error("Server error", "error", err)
		os.Exit(1)
	}

	utils.Info("Shutting down Star Rail Guide MCP Server...")
}