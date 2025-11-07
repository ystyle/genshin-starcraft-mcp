package scraper

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"genshin-starcraft-mcp/pkg/models"
	"genshin-starcraft-mcp/pkg/utils"
)

// 全局正则表达式，避免重复编译
var nodeNameRegex = regexp.MustCompile(`^\d+\.\s*`)

// 全局节点类型映射表
var nodeTypeMap = map[string]map[string]string{
	"服务器节点": {
		"执行节点":       "mhw66orrrfkm",  // 服务器执行节点
		"事件节点":       "mhn7ko01v3yw",  // 服务器事件节点
		"流程控制节点":     "mhe8yn9bysd6",  // 服务器流程控制节点
		"查询节点":       "mhwbqlrw655q",  // 服务器查询节点
		"运算节点":       "mhnd4l069tk0",  // 服务器运算节点
	},
	"客户端节点": {
		"查询节点":       "mholjx05ji8w",  // 客户端查询节点
		"运算节点":       "mhfmxw9fn6n6",  // 客户端运算节点
		"执行节点":       "mh6obvipqv1g",  // 客户端执行节点
		"流程控制节点":     "mhxppurzujfq",  // 客户端流程控制节点
		"其它节点":       "mhor3u09y7u0",  // 客户端其它节点
	},
}


// GetNodeGraphs 获取节点图列表
func (b *Browser) GetNodeGraphs(clientType string, nodeType string) ([]models.NodeGraphItem, error) {
	utils.Debug("Getting node graphs", "client_type", clientType, "node_type", nodeType)

	// 获取完整的页面数据（包含所有节点详情）
	pageData, err := b.getNodeGraphPageData(clientType, nodeType)
	if err != nil {
		utils.Error("Failed to get node graph page data", "client_type", clientType, "node_type", nodeType, "error", err)
		return nil, err
	}

	// 从完整数据中提取节点名称列表，保持顺序
	var nodeGraphs []models.NodeGraphItem
	for _, nodeDetails := range pageData.Nodes {
		// 使用nodeDetails中已经包含的完整分类信息（包含h1分类）
		nodeGraphs = append(nodeGraphs, models.NodeGraphItem{
			Name:        nodeDetails.NodeName,
			Description: nodeDetails.Description, // 返回节点描述
			Category:    nodeDetails.NodeType, // 使用包含h1分类的完整NodeType
		})
	}

	utils.Debug("Retrieved node graph list", "client_type", clientType, "node_type", nodeType, "count", len(nodeGraphs))
	utils.Info("Successfully retrieved node graph list", "client_type", clientType, "node_type", nodeType, "nodes_count", len(nodeGraphs))
	return nodeGraphs, nil
}

// GetNodeGraphDetails 获取节点图详细信息
func (b *Browser) GetNodeGraphDetails(clientType string, nodeType string, nodeName string) (*models.NodeGraphDetails, error) {
	utils.Debug("Getting node graph details", "client_type", clientType, "node_type", nodeType, "node_name", nodeName)

	// 获取完整的页面数据
	pageData, err := b.getNodeGraphPageData(clientType, nodeType)
	if err != nil {
		utils.Error("Failed to get node graph page data", "client_type", clientType, "node_type", nodeType, "error", err)
		return nil, err
	}

	// 调试：打印前几个节点的信息以检查匹配
	for i := 0; i < len(pageData.Nodes) && i < 5; i++ {
		node := pageData.Nodes[i]
		utils.Debug("Available node", "index", i, "name", node.NodeName, "description", node.Description)
	}

	// pageData 已经在上面获取了，这里不需要重复获取

	// 查找指定的节点
	for _, nodeDetails := range pageData.Nodes {
		if nodeDetails.NodeName == nodeName {
			utils.Debug("Found node details", "node_name", nodeName, "description_length", len(nodeDetails.Description), "parameters_count", len(nodeDetails.Parameters), "inputs_count", len(nodeDetails.Inputs), "outputs_count", len(nodeDetails.Outputs))
			utils.Info("Successfully retrieved node details", "node_name", nodeName, "client_type", clientType, "node_type", nodeType)
			return nodeDetails, nil
		}
	}

	utils.Error("Node not found in page data", "client_type", clientType, "node_type", nodeType, "node_name", nodeName, "available_nodes", len(pageData.Nodes))
	return nil, fmt.Errorf("node not found: %s", nodeName)
}

// getNodeGraphPageData 获取节点图页面数据（共享的解析和缓存逻辑）
func (b *Browser) getNodeGraphPageData(clientType string, nodeType string) (*models.NodeGraphPage, error) {
	// 生成缓存key
	cacheKey := fmt.Sprintf("%s_%s", clientType, nodeType)
	utils.Debug("Getting node graph page data", "cache_key", cacheKey)

	// 检查缓存
	if cachedPage, exists := b.nodeGraphCache[cacheKey]; exists {
		utils.Debug("Using cached node graph page", "cache_key", cacheKey, "count", len(cachedPage.Nodes), "last_updated", cachedPage.LastUpdated)
		return cachedPage, nil
	}
	utils.Debug("Cache miss, fetching fresh data", "cache_key", cacheKey)

	// 根据客户端类型和节点类型获取对应的ID
	graphID := b.getNodeGraphID(clientType, nodeType)
	if graphID == "" {
		// 提供更友好的错误信息，包含支持的node_type列表
		supportedTypes := b.getSupportedNodeTypes(clientType)

		errMsg := fmt.Sprintf("invalid node type '%s' for client_type '%s'. Supported node types: %v", nodeType, clientType, supportedTypes)
		utils.Error("Failed to get node graph ID", "client_type", clientType, "node_type", nodeType, "supported_types", supportedTypes)
		return nil, fmt.Errorf(errMsg)
	}

	utils.Debug("Creating page for node graph", "graph_id", graphID)

	// 获取页面内容
	pageURL := fmt.Sprintf("https://act.mihoyo.com/ys/ugc/tutorial/detail/%s", graphID)
	utils.Debug("Creating page with URL", "url", pageURL, "graph_id", graphID)
	page, err := b.NewPage(pageURL)
	if err != nil {
		utils.Error("Failed to create page", "graph_id", graphID, "url", pageURL, "error", err)
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	utils.Debug("Page created, waiting for load...", "graph_id", graphID)

	// 等待页面加载完成
	page.MustWaitLoad()
	utils.Debug("Page loaded, checking content...", "graph_id", graphID)

	// 检查页面是否成功加载
	pageInfo, err := page.Info()
	if err != nil {
		utils.Error("Failed to get page info after load", "graph_id", graphID, "error", err)
		return nil, fmt.Errorf("failed to get page info after load: %w", err)
	}
	utils.Debug("Page loaded successfully", "graph_id", graphID, "url", pageInfo.URL, "title", pageInfo.Title)

	// 等待主要内容区域加载
	utils.Debug("Waiting for main content...", "graph_id", graphID)
	_, err = page.Timeout(pageLoadTimeout).Element("div.doc-view")
	if err != nil {
		utils.Debug("Main content not found", "graph_id", graphID, "error", err)
		// 继续尝试解析，可能页面结构不同
	} else {
		utils.Debug("Main content found", "graph_id", graphID)
	}

	// 再次检查页面内容状态
	utils.Debug("Checking page content status...", "graph_id", graphID)

	utils.Debug("Starting page parsing...", "graph_id", graphID)

	// 解析完整的页面结构，包括所有节点的详细信息
	utils.Debug("Starting complete page parsing...", "graph_id", graphID)
	pageData, err := b.parseCompleteNodeGraphPage(page, clientType, nodeType)
	if err != nil {
		utils.Error("Failed to parse page", "graph_id", graphID, "error", err)
		return nil, fmt.Errorf("failed to parse page: %w", err)
	}
	utils.Debug("Page parsing completed successfully", "graph_id", graphID, "nodes_count", len(pageData.Nodes))

	// 缓存完整的页面数据
	if len(pageData.Nodes) > 0 {
		b.nodeGraphCache[cacheKey] = pageData
		utils.Debug("Cached complete node graph page", "cache_key", cacheKey, "count", len(pageData.Nodes), "last_updated", pageData.LastUpdated)
		utils.Info("Successfully cached node graph page", "cache_key", cacheKey, "nodes_count", len(pageData.Nodes))
	} else {
		utils.Debug("No nodes found in page, skipping cache", "cache_key", cacheKey, "graph_id", graphID)
	}

	return pageData, nil
}


// getSupportedNodeTypes 获取支持的节点类型列表
func (b *Browser) getSupportedNodeTypes(clientType string) []string {
	var supportedTypes []string
	if clientTypes, ok := nodeTypeMap[clientType]; ok {
		for nodeType := range clientTypes {
			supportedTypes = append(supportedTypes, nodeType)
		}
	}
	return supportedTypes
}

// getNodeGraphID 根据客户端类型和节点类型获取对应的页面ID
func (b *Browser) getNodeGraphID(clientType string, nodeType string) string {
	if clientTypes, ok := nodeTypeMap[clientType]; ok {
		if graphID, ok := clientTypes[nodeType]; ok {
			return graphID
		}
	}
	return ""
}



// parseCompleteNodeGraphPage 解析完整的节点图页面，包括所有节点的详细信息
func (b *Browser) parseCompleteNodeGraphPage(page *rod.Page, clientType string, nodeType string) (*models.NodeGraphPage, error) {
	utils.Debug("Starting optimized node graph page parsing", "client_type", clientType, "node_type", nodeType)

	pageData := &models.NodeGraphPage{
		ClientType: clientType,
		NodeType:   nodeType,
		Nodes:      []*models.NodeGraphDetails{},
		LastUpdated: time.Now(),
	}

	// 使用优化的一次性解析所有节点的详细信息
	pageData.Nodes = b.parseAllNodeDetails(page, clientType, nodeType)

	utils.Debug("Parsed complete node graph page", "total_nodes", len(pageData.Nodes), "client_type", clientType, "node_type", nodeType)

	if len(pageData.Nodes) == 0 {
		utils.Debug("No valid nodes found after parsing", "client_type", clientType, "node_type", nodeType)
	}

	return pageData, nil
}

// getSiblingsUntilNextH2 获取从指定元素开始的所有兄弟元素，直到遇到下一个h2
func (b *Browser) getSiblingsUntilNextH2(startElement *rod.Element) []*rod.Element {
	var siblings []*rod.Element
	current := startElement

	for {
		next, err := current.Next()
		if err != nil || next == nil {
			break
		}

		// 如果遇到下一个h2，停止
		if next.MustMatches("h2") {
			break
		}

		siblings = append(siblings, next)
		current = next
	}

	return siblings
}







// NodeBuilder 流式节点构建器
type NodeBuilder struct {
	name        string
	description strings.Builder
	parameters  []models.Param
	inputs      []models.Param
	outputs     []models.Param
	example     strings.Builder
	hasTable    bool
}

// NewNodeBuilder 创建新的节点构建器
func (b *Browser) NewNodeBuilder(h2Element *rod.Element) *NodeBuilder {
	rawName := strings.TrimSpace(h2Element.MustText())
	nodeName := b.cleanNodeName(rawName)

	utils.Debug("Starting new node", "original_name", rawName, "cleaned_name", nodeName)

	return &NodeBuilder{
		name:     nodeName,
		hasTable: false,
	}
}

// AddContent 添加内容到当前节点
func (nb *NodeBuilder) AddContent(element *rod.Element) {
	if nb == nil {
		return
	}

	// 简化元素识别
	elementTag := "unknown"
	if element.MustMatches("p") {
		elementTag = "p"
	} else if element.MustMatches("div.table-wrapper") {
		elementTag = "table-wrapper"
	} else {
		elementTag = "other"
	}

	text := strings.TrimSpace(element.MustText())
	if len(text) > 50 {
		text = text[:50] + "..."
	}

	utils.Debug("Processing element",
		"node_name", nb.name,
		"element_tag", elementTag,
		"text", text,
		"has_table", nb.hasTable)

	switch elementTag {
	case "p":
		text := strings.TrimSpace(element.MustText())
		if text != "" {
			// 检查是否是示例内容
			if nb.hasTable && (strings.Contains(text, "示例") || strings.Contains(text, "用法")) {
				nb.example.WriteString(text)
				nb.example.WriteString("\n")
			} else if strings.Contains(text, "节点功能") ||
					  strings.Contains(text, "节点参数") ||
					  strings.Contains(text, "参数类型") {
				// 跳过这些标题元素
				utils.Debug("Skipping title element", "node_name", nb.name, "text", text)
			} else if nb.description.Len() == 0 {
				// 第一个非标题的p元素作为描述
				nb.description.WriteString(text)
				utils.Debug("Found description", "node_name", nb.name, "description", text)
			}
		}

	case "table-wrapper":
		utils.Debug("Found table-wrapper element", "node_name", nb.name)
		nb.parseTable(element)
}
}

// parseTable 解析表格内容
func (nb *NodeBuilder) parseTable(divWrapperElement *rod.Element) {
	// 在div.table-wrapper内部查找table元素
	tableElement, err := divWrapperElement.Element("table")
	if err != nil {
		utils.Debug("Failed to find table element in div wrapper", "error", err)
		return
	}

	// 使用CSS选择器获取表格行
	tableRows, err := tableElement.Elements("tbody tr")
	if err != nil {
		utils.Debug("Failed to parse table with tbody tr", "error", err)
		// 如果没有tbody，尝试直接获取tr
		tableRows, err = tableElement.Elements("tr")
		if err != nil {
			utils.Debug("Failed to parse table with tr", "error", err)
			return
		}
	}

	utils.Debug("Parsing table with CSS selectors", "node_name", nb.name, "total_rows", len(tableRows))

	// 遍历所有行
	for i, row := range tableRows {
		// 使用CSS选择器获取单元格
		cells, err := row.Elements("td")
		if err != nil {
			utils.Debug("Failed to get cells for row", "node_name", nb.name, "row", i, "error", err)
			continue
		}

		// 确保有足够的列（参数类型、参数名、类型、说明）
		if len(cells) < 4 {
			utils.Debug("Skipping row with insufficient columns", "node_name", nb.name, "row", i, "columns", len(cells))
			continue
		}

		// 直接获取单元格内容
		paramType := strings.TrimSpace(cells[0].MustText())   // 参数类型（入参/出参）
		paramName := strings.TrimSpace(cells[1].MustText())   // 参数名
		dataType := strings.TrimSpace(cells[2].MustText())   // 类型
		description := strings.TrimSpace(cells[3].MustText()) // 说明

		utils.Debug("Parsing table row with CSS", "node_name", nb.name, "row", i, "param_type", paramType, "param_name", paramName)

		// 跳过空行或表头行
		if paramType == "" && paramName == "" && dataType == "" && description == "" {
			utils.Debug("Skipping empty row", "node_name", nb.name, "row", i)
			continue
		}

		// 跳过表头行（包含"参数类型"、"参数名"等）
		if strings.Contains(paramType, "参数类型") || strings.Contains(paramName, "参数名") ||
		   strings.Contains(dataType, "类型") || strings.Contains(description, "说明") {
			utils.Debug("Skipping header row", "node_name", nb.name, "row", i)
			continue
		}

		// 创建参数对象
		param := models.Param{
			Name:        paramName,
			Type:        dataType,
			Description: description,
			Required:    true,
		}

		// 根据参数类型分类
		if paramType == "入参" {
			nb.inputs = append(nb.inputs, param)
			utils.Debug("Added input parameter", "node_name", nb.name, "param", paramName, "type", dataType)
		} else if paramType == "出参" {
			nb.outputs = append(nb.outputs, param)
			utils.Debug("Added output parameter", "node_name", nb.name, "param", paramName, "type", dataType)
		} else {
			nb.parameters = append(nb.parameters, param)
			utils.Debug("Added other parameter", "node_name", nb.name, "param", paramName, "type", dataType)
		}
	}
}

// Build 构建最终的节点详情
func (nb *NodeBuilder) Build() *models.NodeGraphDetails {
	if nb == nil {
		return nil
	}

	nodeDetails := &models.NodeGraphDetails{
		NodeName:     nb.name,
		Description:  nb.description.String(),
		ClientType:   "", // 将在调用方设置
		NodeType:     "", // 将在调用方设置
		Parameters:   nb.parameters,
		Inputs:       nb.inputs,
		Outputs:      nb.outputs,
		Example:      strings.TrimSpace(nb.example.String()),
		LastUpdated:  time.Now(),
	}

	utils.Debug("Built node", "node_name", nb.name, "description_length", len(nodeDetails.Description), "parameters_count", len(nb.parameters), "inputs_count", len(nb.inputs), "outputs_count", len(nb.outputs))

	return nodeDetails
}

// parseAllNodeDetails 一次性解析所有节点的详细信息，使用流式算法避免O(n²)复杂度
func (b *Browser) parseAllNodeDetails(page *rod.Page, clientType string, nodeType string) []*models.NodeGraphDetails {
	utils.Debug("Starting optimized node graph page parsing", "client_type", clientType, "node_type", nodeType)

	// 获取所有h1和h2元素
	h1Elements, err := page.Elements("h1")
	if err != nil {
		utils.Error("Failed to get h1 elements", "error", err)
		return nil
	}

	h2Elements, err := page.Elements("h2")
	if err != nil {
		utils.Error("Failed to get h2 elements", "error", err)
		return nil
	}

	utils.Debug("Found elements for streaming parsing", "h1_count", len(h1Elements), "h2_count", len(h2Elements))

	// 构建h1分类映射：h2的文本对应的h1分类
	h2ToH1Category := make(map[string]string)
	currentH1Category := "未分类"

	// 遍历所有元素来建立分类映射
	allElements, err := page.Elements("h1, h2")
	if err != nil {
		utils.Error("Failed to get all elements", "error", err)
		return nil
	}

	for _, element := range allElements {
		if element.MustMatches("h1") {
			// 更新当前h1分类
			currentH1Category = strings.TrimSpace(element.MustText())
			utils.Debug("Found h1 category", "category", currentH1Category)
		} else if element.MustMatches("h2") {
			// 为h2元素分配当前的h1分类
			h2Text := strings.TrimSpace(element.MustText())
			h2ToH1Category[h2Text] = currentH1Category
			utils.Debug("Mapped h2 to h1 category", "h2", h2Text, "h1_category", currentH1Category)
		}
	}

	// 调试：打印映射关系
	utils.Debug("H2 to H1 mapping summary", "total_mappings", len(h2ToH1Category))
	for h2, h1 := range h2ToH1Category {
		if len(h1) > 0 && h1 != "未分类" {
			utils.Debug("Sample mapping", "h2", h2, "h1", h1)
			break // 只打印第一个有效映射作为样本
		}
	}

	var nodes []*models.NodeGraphDetails

	// 流式处理：对每个h2元素，直接处理其后续兄弟元素
	for i, h2Element := range h2Elements {
		// 获取节点名称
		rawName := strings.TrimSpace(h2Element.MustText())
		nodeName := b.cleanNodeName(rawName)
		h1Category := h2ToH1Category[rawName] // 获取对应的h1分类

		utils.Debug("Processing h2 element", "index", i, "node_name", nodeName, "h1_category", h1Category)

		// 跳过空标题
		if nodeName == "" {
			utils.Debug("Empty title, skipping", "index", i)
			continue
		}

		// 创建节点构建器
		nodeBuilder := &NodeBuilder{
			name:     nodeName,
			hasTable: false,
		}

		// 获取h2后的所有兄弟元素，直到下一个h2
		siblings := b.getSiblingsUntilNextH2(h2Element)
		utils.Debug("Found siblings for node", "node_name", nodeName, "siblings_count", len(siblings))

		// 处理兄弟元素
		for _, sibling := range siblings {
			nodeBuilder.AddContent(sibling)
		}

		// 构建节点详情
		nodeDetails := nodeBuilder.Build()
		if nodeDetails != nil {
			nodeDetails.ClientType = clientType
			nodeDetails.NodeType = nodeType
			// 在NodeType中包含h1分类信息
			if h1Category != "" {
				nodeDetails.NodeType = fmt.Sprintf("%s - %s", h1Category, nodeType)
			}
			nodes = append(nodes, nodeDetails)
			utils.Debug("Added node to list", "index", i, "name", nodeName, "h1_category", h1Category, "description_length", len(nodeDetails.Description), "parameters_count", len(nodeDetails.Parameters), "inputs_count", len(nodeDetails.Inputs), "outputs_count", len(nodeDetails.Outputs))
		}
	}

	utils.Debug("Completed streaming parsing", "total_nodes", len(nodes), "client_type", clientType, "node_type", nodeType)
	return nodes
}

// cleanNodeName 清理节点名称，去掉开头的数字序号和特殊字符
func (b *Browser) cleanNodeName(nodeName string) string {
	// 使用全局正则表达式匹配并去掉开头的数字序号，如 "1. ", "2. ", "10. " 等
	cleaned := nodeNameRegex.ReplaceAllString(nodeName, "")

	// 去除首尾空格
	cleaned = strings.TrimSpace(cleaned)

	utils.Debug("Cleaned node name", "original", nodeName, "cleaned", cleaned)
	return cleaned
}