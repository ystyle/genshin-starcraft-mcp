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
		nodeGraphs = append(nodeGraphs, models.NodeGraphItem{
			Name:        nodeDetails.NodeName,
			Description: "", // 列表不返回描述
			Category:    nodeType, // 使用节点类型作为分类
		})
	}

	utils.Debug("Retrieved node graph list", "client_type", clientType, "node_type", nodeType, "count", len(nodeGraphs))
	utils.Info("Successfully retrieved node graph list", "client_type", clientType, "node_type", nodeType, "nodes_count", len(nodeGraphs))
	return nodeGraphs, nil
}

// GetNodeGraphDetails 获取节点图详细信息
func (b *Browser) GetNodeGraphDetails(clientType string, nodeType string, nodeName string) (*models.NodeGraphDetails, error) {
	utils.Debug("Getting node graph details", "client_type", clientType, "node_type", nodeType, "node_name", nodeName)

	// 获取完整的页面数据（包含所有节点详情）
	pageData, err := b.getNodeGraphPageData(clientType, nodeType)
	if err != nil {
		utils.Error("Failed to get node graph page data for details", "client_type", clientType, "node_type", nodeType, "node_name", nodeName, "error", err)
		return nil, err
	}

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
		errMsg := fmt.Sprintf("invalid client type or node type. client_type: %s, node_type: %s", clientType, nodeType)
		utils.Error("Failed to get node graph ID", "client_type", clientType, "node_type", nodeType)
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


// getNodeGraphID 根据客户端类型和节点类型获取对应的页面ID
func (b *Browser) getNodeGraphID(clientType string, nodeType string) string {
	// 根据导航列表映射客户端类型和节点类型到页面ID
	nodeTypeMap := map[string]map[string]string{
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

	if clientTypes, ok := nodeTypeMap[clientType]; ok {
		if graphID, ok := clientTypes[nodeType]; ok {
			return graphID
		}
	}

	return ""
}



// parseCompleteNodeGraphPage 解析完整的节点图页面，包括所有节点的详细信息
func (b *Browser) parseCompleteNodeGraphPage(page *rod.Page, clientType string, nodeType string) (*models.NodeGraphPage, error) {
	utils.Debug("Starting complete node graph page parsing", "client_type", clientType, "node_type", nodeType)

	// 查找所有h2标题元素（节点标题）
	h2Elements, err := page.Elements("h2")
	if err != nil {
		utils.Error("Failed to find h2 elements", "client_type", clientType, "node_type", nodeType, "error", err)
		return nil, fmt.Errorf("failed to find h2 elements: %w", err)
	}

	utils.Debug("Found h2 elements for complete parsing", "count", len(h2Elements), "client_type", clientType, "node_type", nodeType)

	if len(h2Elements) == 0 {
		utils.Debug("No h2 elements found in page", "client_type", clientType, "node_type", nodeType)
	}

	pageData := &models.NodeGraphPage{
		ClientType: clientType,
		NodeType:   nodeType,
		Nodes:      []*models.NodeGraphDetails{},
		LastUpdated: time.Now(),
	}

	// 一次性解析所有节点的详细信息
	pageData.Nodes = b.parseAllNodeDetails(page, h2Elements, clientType, nodeType)

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

// extractDescriptionFromSiblings 从兄弟元素中提取节点描述
func (b *Browser) extractDescriptionFromSiblings(siblings []*rod.Element) string {
	for _, sibling := range siblings {
		if sibling.MustMatches("p") {
			text := strings.TrimSpace(sibling.MustText())
			if text != "" &&
			   !strings.Contains(text, "节点参数") &&
			   !strings.Contains(text, "参数类型") &&
			   !strings.Contains(text, "节点功能") {
				return text
			}
		}
	}
	return ""
}

// parseAllNodeDetails 一次性解析所有节点的详细信息
func (b *Browser) parseAllNodeDetails(page *rod.Page, h2Elements []*rod.Element, clientType string, nodeType string) []*models.NodeGraphDetails {
	var allNodes []*models.NodeGraphDetails

	for i, h2Element := range h2Elements {
		// 获取h2文本内容作为节点名称，并清理开头的数字序号
		rawNodeName := strings.TrimSpace(h2Element.MustText())
		nodeName := b.cleanNodeName(rawNodeName)
		utils.Debug("Processing h2 element", "index", i, "node_name", nodeName)

		// 简单的过滤：只跳过空标题
		if nodeName == "" {
			utils.Debug("Empty title, skipping", "index", i)
			continue
		}

		// 获取h2后的所有兄弟元素，直到下一个h2
		siblings := b.getSiblingsUntilNextH2(h2Element)
		utils.Debug("Found siblings for node", "node_name", nodeName, "siblings_count", len(siblings))

		// 从兄弟元素中提取描述
		description := b.extractDescriptionFromSiblings(siblings)

		// 使用高性能解析函数获取完整的节点详情，包括参数、输入、输出、示例
		// 传递h2Elements避免重复DOM查询
		parameters, inputs, outputs, example := b.parseNodeGraphDetails(page, nodeName, h2Elements)

		nodeDetails := &models.NodeGraphDetails{
			NodeName:     nodeName,
			Description:  description,
			ClientType:   clientType,
			NodeType:     nodeType,
			Parameters:   parameters,
			Inputs:       inputs,
			Outputs:      outputs,
			Example:      example,
			LastUpdated:  time.Now(),
		}

		allNodes = append(allNodes, nodeDetails)
		utils.Debug("Added node to list", "index", i, "name", nodeName, "description_length", len(nodeDetails.Description), "parameters_count", len(parameters), "inputs_count", len(inputs), "outputs_count", len(outputs))
	}

	return allNodes
}




// parseNodeGraphDetails 解析节点图详细信息，包括参数表格
func (b *Browser) parseNodeGraphDetails(page *rod.Page, nodeName string, h2Elements []*rod.Element) ([]models.Param, []models.Param, []models.Param, string) {
	var parameters, inputs, outputs []models.Param
	var example string

	// 使用预获取的h2元素列表来查找目标节点，避免重复DOM查询
	utils.Debug("parseNodeGraphDetails called with pre-fetched h2 elements", "node_name", nodeName, "h2_elements_count", len(h2Elements))

	// 找到匹配的节点
	var targetH2 *rod.Element
	for _, h2Element := range h2Elements {
		h2Text := strings.TrimSpace(h2Element.MustText())
		if h2Text == nodeName {
			targetH2 = h2Element
			break
		}
	}

	if targetH2 == nil {
		utils.Debug("Node not found in page", "node_name", nodeName)
		return parameters, inputs, outputs, example
	}

	// 使用高效的兄弟元素遍历算法，避免全元素遍历
	siblings := b.getSiblingsUntilNextH2(targetH2)
	utils.Debug("Found siblings for node details", "node_name", nodeName, "siblings_count", len(siblings))

	// 遍历兄弟元素查找表格和示例
	foundTable := false
	var exampleText strings.Builder

	for _, sibling := range siblings {
		siblingText := strings.TrimSpace(sibling.MustText())

		// 查找表格
		if sibling.MustMatches("table") && !foundTable {
			utils.Debug("Found table for node", "node_name", nodeName)
			foundTable = true

			// 解析表格内容
			tableRows, err := sibling.Elements("tr")
			if err != nil {
				utils.Debug("Failed to find table rows", "error", err)
				continue
			}

			// 跳过表头行，从第二行开始
			for i := 1; i < len(tableRows); i++ {
				row := tableRows[i]
				cells, err := row.Elements("td")
				if err != nil || len(cells) < 4 {
					utils.Debug("Table row has insufficient cells", "row", i, "cells", len(cells))
					continue
				}

				// 获取单元格内容
				paramType := strings.TrimSpace(cells[0].MustText())   // 参数类型（入参/出参）
				paramName := strings.TrimSpace(cells[1].MustText())   // 参数名
				dataType := strings.TrimSpace(cells[2].MustText())   // 类型
				description := strings.TrimSpace(cells[3].MustText()) // 说明

				// 跳过空行
				if paramType == "" && paramName == "" && dataType == "" && description == "" {
					continue
				}

				// 创建参数对象
				param := models.Param{
					Name:        paramName,
					Type:        dataType,
					Description: description,
					Required:    true, // 默认为必需
				}

				// 根据参数类型分类
				if paramType == "入参" {
					inputs = append(inputs, param)
				} else if paramType == "出参" {
					outputs = append(outputs, param)
				} else {
					// 其他类型的参数
					parameters = append(parameters, param)
				}
			}
		}

		// 查找使用示例
		if foundTable && (strings.Contains(siblingText, "示例") || strings.Contains(siblingText, "用法")) {
			utils.Debug("Found example section", "node_name", nodeName)
			exampleText.WriteString(siblingText)
			exampleText.WriteString("\n")
		}
	}

	example = strings.TrimSpace(exampleText.String())

	utils.Debug("Parsed node details", "node_name", nodeName, "parameters", len(parameters), "inputs", len(inputs), "outputs", len(outputs))
	return parameters, inputs, outputs, example
}

// cleanNodeName 清理节点名称，去掉开头的数字序号和特殊字符
func (b *Browser) cleanNodeName(nodeName string) string {
	// 使用正则表达式匹配并去掉开头的数字序号，如 "1. ", "2. ", "10. " 等
	re := regexp.MustCompile(`^\d+\.\s*`)
	cleaned := re.ReplaceAllString(nodeName, "")

	// 去除首尾空格
	cleaned = strings.TrimSpace(cleaned)

	utils.Debug("Cleaned node name", "original", nodeName, "cleaned", cleaned)
	return cleaned
}