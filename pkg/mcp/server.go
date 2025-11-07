package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"genshin-starcraft-mcp/pkg/models"
	"genshin-starcraft-mcp/pkg/scraper"
	"genshin-starcraft-mcp/pkg/utils"
)

// GenshinStarcraftMCPServer 使用官方MCP库的服务器
type GenshinStarcraftMCPServer struct {
	browser *scraper.Browser
	server  *server.MCPServer
}

// NewGenshinStarcraftMCPServer 创建新的MCP服务器
func NewGenshinStarcraftMCPServer() (*GenshinStarcraftMCPServer, error) {
	utils.Debug("Creating new MCP server with official library")

	browser, err := scraper.NewBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}

	// 创建MCP服务器
	s := server.NewMCPServer(
		"原神千星奇域教程",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// // 添加搜索工具
	// searchTool := mcp.NewTool("search",
	// 	mcp.WithDescription("在原神千星奇域官方综合指南网站中搜索相关内容。支持搜索节点功能、参数说明、使用方法、配置指南等教程文档。"),
	// 	mcp.WithString("query",
	// 		mcp.Required(),
	// 		mcp.Description("搜索关键词，例如：查询对局游玩方式、获取局部变量、服务器配置等"),
	// 	),
	// )

	// 添加导航工具
	navigationTool := mcp.NewTool("get_navigation",
		mcp.WithDescription("获取原神千星奇域综合指南网站的完整导航目录，返回所有可用的教程分类和章节链接列表。"),
	)

	// 添加指南工具
	tutorialTool := mcp.NewTool("get_guide",
		mcp.WithDescription("根据导航目录中的ID获取具体的教程内容，包括节点功能说明、参数表格、使用方法、配置说明等详细信息。"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("教程页面ID，例如'mh29wpicgvh0'，从get_navigation工具返回的导航列表中获取"),
		),
	)

	// // 添加打开搜索结果工具
	// openSearchTool := mcp.NewTool("open_search_result",
	// 	mcp.WithDescription("根据搜索结果中的标题直接打开对应的教程页面，获取完整的教程内容。"),
	// 	mcp.WithString("title",
	// 		mcp.Required(),
	// 		mcp.Description("要打开的搜索结果标题，从search工具返回的搜索结果列表中选择"),
	// 	),
	// )

	// 添加获取节点图列表工具
	nodeGraphsTool := mcp.NewTool("get_node_graphs",
		mcp.WithDescription("获取指定类型的千星奇域节点图列表，返回该类型下所有可用的节点名称和功能描述。结果会被缓存以提高查询效率。"),
		mcp.WithString("client_type",
			mcp.Required(),
			mcp.Description("客户端类型：'客户端节点'或 '服务器节点'"),
		),
		mcp.WithString("node_type",
			mcp.Required(),
			mcp.Description("节点类型。客户端节点可选：查询节点/运算节点/执行节点/流程控制节点/其它节点；服务器节点可选：执行节点/事件节点/流程控制节点/查询节点/运算节点"),
		),
	)

	// 添加获取节点图详情工具
	nodeGraphDetailsTool := mcp.NewTool("get_node_graph_details",
		mcp.WithDescription("获取指定节点的详细信息，包括参数表格、输入输出说明、使用示例等。会利用get_node_graphs工具的缓存数据提高效率。"),
		mcp.WithString("client_type",
			mcp.Required(),
			mcp.Description("客户端类型：'客户端节点'或 '服务器节点'，用于准确定位节点"),
		),
		mcp.WithString("node_type",
			mcp.Required(),
			mcp.Description("节点类型，用于准确定位节点。客户端节点可选：查询节点/运算节点/执行节点/流程控制节点/其它节点；服务器节点可选：执行节点/事件节点/流程控制节点/查询节点/运算节点"),
		),
		mcp.WithString("node_name",
			mcp.Required(),
			mcp.Description("节点的完整名称，从get_node_graphs工具返回的节点列表中选择，例如'查询对局游玩方式及人数'"),
		),
	)

	genshinServer := &GenshinStarcraftMCPServer{
		browser: browser,
		server:  s,
	}

	// 添加工具处理器
	// s.AddTool(searchTool, genshinServer.handleSearch)
	s.AddTool(navigationTool, genshinServer.handleGetNavigation)
	s.AddTool(tutorialTool, genshinServer.handleGetGuide)
	// s.AddTool(openSearchTool, genshinServer.handleOpenSearchResult)
	s.AddTool(nodeGraphsTool, genshinServer.handleGetNodeGraphs)
	s.AddTool(nodeGraphDetailsTool, genshinServer.handleGetNodeGraphDetails)

	utils.Debug("MCP server created successfully with official library")
	return genshinServer, nil
}

// Close 关闭服务器
func (s *GenshinStarcraftMCPServer) Close() error {
	utils.Debug("Closing MCP server")
	if s.browser != nil {
		return s.browser.Close()
	}
	return nil
}

// Start 启动MCP服务器
func (s *GenshinStarcraftMCPServer) Start() error {
	utils.Debug("Starting MCP server with official library")
	return server.ServeStdio(s.server)
}

// handleSearch 处理搜索请求
func (s *GenshinStarcraftMCPServer) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	utils.Debug("Handling search", "query", query)

	searchID, results, err := s.browser.Search(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	content := fmt.Sprintf("搜索ID: %s\n找到 %d 个搜索结果", searchID, len(results))
	if len(results) > 0 {
		content += "\n\n"
		for i, result := range results {
			content += fmt.Sprintf("%d. [%s](%s)\n   %s\n", i+1, result.Title, result.URL, result.Description)
		}
	}

	return mcp.NewToolResultText(content), nil
}

// handleGetNavigation 处理获取导航请求
func (s *GenshinStarcraftMCPServer) handleGetNavigation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	utils.Debug("Handling get_navigation")

	items, err := s.browser.GetNavigation()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取导航失败: %v", err)), nil
	}

	if len(items) == 0 {
		return mcp.NewToolResultText("没有找到导航目录"), nil
	}

	var navText string
	navText = "导航目录:\n\n"
	for i, item := range items {
		navText += fmt.Sprintf("%d. [%s](%s)\n", i+1, item.Title, item.URL)
	}

	return mcp.NewToolResultText(navText), nil
}

// handleGetGuide 处理获取指南请求
func (s *GenshinStarcraftMCPServer) handleGetGuide(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	utils.Debug("Handling get_guide", "id", id)

	tutorial, err := s.browser.GetTutorial(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取指南失败: %v", err)), nil
	}

	fullURL := fmt.Sprintf("https://act.mihoyo.com/ys/ugc/tutorial/detail/%s", tutorial.URL)
	content := fmt.Sprintf("# %s\n\n%s\n\n[原文链接](%s)",
		tutorial.Title, tutorial.Content, fullURL)

	return mcp.NewToolResultText(content), nil
}

// handleOpenSearchResult 处理打开搜索结果请求
func (s *GenshinStarcraftMCPServer) handleOpenSearchResult(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	utils.Debug("Handling open_search_result", "title", title)

	content, err := s.browser.OpenSearchResultByTitle(title)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("打开搜索结果失败: %v", err)), nil
	}

	// 直接返回页面内容，不需要额外的格式化
	responseContent := fmt.Sprintf("# 搜索结果：%s\n\n%s", title, content)

	return mcp.NewToolResultText(responseContent), nil
}

// handleGetNodeGraphs 处理获取节点图列表请求
func (s *GenshinStarcraftMCPServer) handleGetNodeGraphs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clientType, err := request.RequireString("client_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nodeType, err := request.RequireString("node_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	utils.Debug("Handling get_node_graphs", "client_type", clientType, "node_type", nodeType)

	nodeGraphs, err := s.browser.GetNodeGraphs(clientType, nodeType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取节点图列表失败: %v", err)), nil
	}

	// 调试：打印前几个节点的分类信息
	for i := 0; i < len(nodeGraphs) && i < 5; i++ {
		node := nodeGraphs[i]
		utils.Debug("Node info", "index", i, "name", node.Name, "category", node.Category, "description", node.Description)
	}

	// 格式化返回内容，按h1分类分组显示
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# 节点图列表 (%s - %s)\n\n", clientType, nodeType))

	if len(nodeGraphs) == 0 {
		content.WriteString("未找到相关节点图\n")
	} else {
		// 按分类分组节点
		categoryGroups := make(map[string][]models.NodeGraphItem)
		var categories []string

		for _, node := range nodeGraphs {
			category := node.Category
			if category == "" {
				category = "未分类"
			}

			// 如果是新的分类，添加到分类列表
			found := false
			for _, existingCategory := range categories {
				if existingCategory == category {
					found = true
					break
				}
			}
			if !found {
				categories = append(categories, category)
			}

			categoryGroups[category] = append(categoryGroups[category], node)
		}

		// 按分类顺序输出节点
		for _, category := range categories {
			// 去掉分类中的 " - 查询节点" 后缀
			cleanCategory := strings.Replace(category, " - 查询节点", "", -1)
			content.WriteString(fmt.Sprintf("- **%s**\n", cleanCategory))
			nodes := categoryGroups[category]
			for _, node := range nodes {
				content.WriteString(fmt.Sprintf("  - **%s**\n", node.Name))
			}
			content.WriteString("\n")
		}
	}

	return mcp.NewToolResultText(content.String()), nil
}

// handleGetNodeGraphDetails 处理获取节点图详情请求
func (s *GenshinStarcraftMCPServer) handleGetNodeGraphDetails(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clientType, err := request.RequireString("client_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nodeType, err := request.RequireString("node_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nodeName, err := request.RequireString("node_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	utils.Debug("Handling get_node_graph_details", "client_type", clientType, "node_type", nodeType, "node_name", nodeName)

	details, err := s.browser.GetNodeGraphDetails(clientType, nodeType, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取节点图详情失败: %v", err)), nil
	}

	// 格式化返回内容
	content := fmt.Sprintf("# %s\n\n**描述**: %s\n\n", details.NodeName, details.Description)

	// 创建markdown表格
	if len(details.Inputs) > 0 || len(details.Outputs) > 0 {
		content += "**参数表格**:\n\n"
		content += "| 参数类型 | 参数名 | 类型 | 说明 |\n"
		content += "|---------|--------|------|------|\n"

		// 先添加入参
		for _, input := range details.Inputs {
			content += fmt.Sprintf("| 入参 | **%s** | %s | %s |\n", input.Name, input.Type, input.Description)
		}

		// 再添加出参
		for _, output := range details.Outputs {
			content += fmt.Sprintf("| 出参 | **%s** | %s | %s |\n", output.Name, output.Type, output.Description)
		}

		// 最后添加其他参数
		for _, param := range details.Parameters {
			content += fmt.Sprintf("| 其他 | **%s** | %s | %s |\n", param.Name, param.Type, param.Description)
		}

		content += "\n"
	}

	if details.Example != "" {
		content += fmt.Sprintf("**使用示例**:\n```%s```\n\n", details.Example)
	}

	return mcp.NewToolResultText(content), nil
}