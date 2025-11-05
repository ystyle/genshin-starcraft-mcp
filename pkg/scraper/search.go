package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"genshin-starcraft-mcp/pkg/models"
	"genshin-starcraft-mcp/pkg/utils"
)

// 统一的等待时间常量
const (
	// 页面加载等待时间
	pageLoadTimeout = 30 * time.Second

	// 元素等待时间
	elementTimeout = 15 * time.Second

	// 快速元素等待时间
	quickElementTimeout = 5 * time.Second

	// 导航等待时间
	navigationTimeout = 20 * time.Second

	// 搜索相关等待时间
	searchBoxTimeout = 15 * time.Second
	searchDialogTimeout = 20 * time.Second

	// 固定延迟
	initialDelay = 3 * time.Second
	shortDelay = 500 * time.Millisecond
	mediumDelay = 2 * time.Second
)

// Search 执行搜索，返回搜索会话ID和结果
func (b *Browser) Search(query string) (string, []models.SearchResult, error) {
	utils.Debug("Starting search", "query", query)

	// 生成唯一的搜索会话ID
	searchID := fmt.Sprintf("search_%d", time.Now().UnixNano())

	// 导航到搜索页面
	page, err := b.NewPage("https://act.mihoyo.com/ys/ugc/tutorial/detail/mh29wpicgvh0")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create search page: %w", err)
	}
	defer page.Close()

	// 等待页面加载完成
	page.MustWaitLoad()

	// 等待页面完全加载并额外等待
	time.Sleep(initialDelay)

	// 等待搜索框出现，增加超时时间
	searchBox, err := page.Timeout(searchBoxTimeout).Element("input[type=search]")
	if err != nil {
		return "", nil, fmt.Errorf("search box not found: %w", err)
	}

	utils.Debug("Found search box, entering query")

	// 清空搜索框并输入查询
	searchBox.MustSelectAllText().MustInput(query)

	// 等待一下再按回车
	time.Sleep(shortDelay)

	// 按回车键
	searchBox.MustKeyActions().Press(input.Enter)

	// 等待对话框出现，增加超时时间和额外等待
	time.Sleep(mediumDelay) // 等待弹窗出现

	dialogElement, err := page.Timeout(searchDialogTimeout).Element("div[role=dialog]")
	if err != nil {
		// 如果找不到弹窗，尝试其他可能的选择器
		utils.Debug("Dialog not found with role=dialog, trying alternatives")
		alternativeSelectors := []string{
			".tw-modal",
			".tw-dialog",
			"[role='dialog']",
			".tw-popup",
			".tw-dropdown",
		}

		for _, selector := range alternativeSelectors {
			dialogElement, err = page.Timeout(searchDialogTimeout).Element(selector)
			if err == nil {
				utils.Debug("Found dialog with alternative selector", "selector", selector)
				break
			}
		}

		if dialogElement == nil {
			return "", nil, fmt.Errorf("search dialog not found with any selector: %w", err)
		}
	}

	utils.Debug("Found search dialog")

	// 等待结果加载，增加等待时间
	time.Sleep(shortDelay)

	// 设置超时上下文获取搜索结果
	resultCtx, resultCancel := context.WithTimeout(context.Background(), quickElementTimeout)
	defer resultCancel()

	// 获取搜索结果
	resultElements, err := dialogElement.Context(resultCtx).Elements("a.tw-relative.tw-block")
	if err != nil {
		return "", nil, fmt.Errorf("failed to find result elements: %w", err)
	}

	utils.Debug("Found result elements", "count", len(resultElements))

	var results []models.SearchResult

	for i, element := range resultElements {
		// 获取标题
		titleElement, err := element.Timeout(quickElementTimeout).Element("div > div > div")
		if err != nil {
			utils.Debug("Failed to get title element", "index", i, "error", err)
			continue
		}
		title := strings.TrimSpace(titleElement.MustText())

		// 获取描述 - 使用最后一个div作为描述
		descElement, err := element.Timeout(quickElementTimeout).Element("div > div > div:last-child")
		if err != nil {
			utils.Debug("Failed to get description element", "index", i, "error", err)
			continue
		}
		description := strings.TrimSpace(descElement.MustText())

		// 搜索结果包含搜索会话ID和结果索引
		result := models.SearchResult{
			Title:       title,
			Description: description,
			URL:         "", // 暂时不获取URL，等AI选择后再获取
			Section:     fmt.Sprintf("%s_result_%d", searchID, i), // 包含搜索ID和结果索引
		}

		results = append(results, result)
		utils.Debug("Added search result", "search_id", searchID, "index", i, "title", title)
	}

	utils.Debug("Search completed", "search_id", searchID, "results", len(results))
	return searchID, results, nil
}

// OpenSearchResult 打开AI选择的搜索结果
func (b *Browser) OpenSearchResult(searchID string, resultIndex int) (string, error) {
	utils.Debug("Opening search result", "search_id", searchID, "index", resultIndex)

	// 导航到教程页面
	page, err := b.NewPage("https://act.mihoyo.com/ys/ugc/tutorial/detail/mh29wpicgvh0")
	if err != nil {
		return "", fmt.Errorf("failed to create search page: %w", err)
	}
	defer page.Close()

	// 等待页面加载完成
	page.MustWaitLoad()

	// 等待搜索框出现
	searchBox, err := b.WaitForElement(page, "input[type=search]", searchBoxTimeout)
	if err != nil {
		return "", fmt.Errorf("search box not found: %w", err)
	}

	// 输入一个通用搜索词触发搜索弹窗
	searchBox.MustInput("教程")

	// 按回车键
	searchBox.MustKeyActions().Press(input.Enter)

	// 等待对话框出现
	dialogElement, err := b.WaitForElement(page, "div[role=dialog]", quickElementTimeout)
	if err != nil {
		return "", fmt.Errorf("search dialog not found: %w", err)
	}

	// 等待结果加载
	page.MustWaitIdle()

	// 获取搜索结果
	resultElements, err := dialogElement.Elements("a.tw-relative.tw-block")
	if err != nil {
		return "", fmt.Errorf("failed to find result elements: %w", err)
	}

	if resultIndex < 0 || resultIndex >= len(resultElements) {
		return "", fmt.Errorf("result index %d out of range (0-%d)", resultIndex, len(resultElements)-1)
	}

	// 点击指定的搜索结果
	selectedElement := resultElements[resultIndex]
	selectedElement.MustClick()

	// 获取新打开页面的URL - 使用浏览器实例的方法
	pages, err := b.browser.Pages()
	if err != nil {
		return "", fmt.Errorf("failed to get pages: %w", err)
	}

	for _, p := range pages {
		if p != page {
			url := p.MustInfo().URL
			utils.Debug("Opened search result", "search_id", searchID, "index", resultIndex, "url", url)
			return url, nil
		}
	}

	return "", fmt.Errorf("failed to get URL for search result %s_%d", searchID, resultIndex)
}

// OpenSearchResultByTitle 根据标题匹配并打开搜索结果，返回页面内容
func (b *Browser) OpenSearchResultByTitle(title string) (string, error) {
	utils.Debug("Opening search result by title", "title", title)

	// 导航到教程页面
	page, err := b.NewPage("https://act.mihoyo.com/ys/ugc/tutorial/detail/mh29wpicgvh0")
	if err != nil {
		return "", fmt.Errorf("failed to create search page: %w", err)
	}
	defer page.Close()

	// 等待页面加载完成
	page.MustWaitLoad()

	// 等待搜索框出现
	searchBox, err := b.WaitForElement(page, "input[type=search]", searchBoxTimeout)
	if err != nil {
		return "", fmt.Errorf("search box not found: %w", err)
	}

	// 输入一个通用搜索词触发搜索弹窗
	searchBox.MustInput("教程")

	// 按回车键
	searchBox.MustKeyActions().Press(input.Enter)

	// 等待对话框出现
	dialogElement, err := b.WaitForElement(page, "div[role=dialog]", quickElementTimeout)
	if err != nil {
		return "", fmt.Errorf("search dialog not found: %w", err)
	}

	// 等待结果加载
	page.MustWaitIdle()

	// 获取搜索结果
	resultElements, err := dialogElement.Elements("a.tw-relative.tw-block")
	if err != nil {
		return "", fmt.Errorf("failed to find result elements: %w", err)
	}

	// 遍历搜索结果，匹配标题
	for i, element := range resultElements {
		// 获取标题
		titleElement, err := element.Timeout(quickElementTimeout).Element("div > div > div")
		if err != nil {
			utils.Debug("Failed to get title element", "index", i, "error", err)
			continue
		}
		currentTitle := strings.TrimSpace(titleElement.MustText())

		// 如果标题匹配，点击这个结果
		if currentTitle == title {
			utils.Debug("Found matching result", "index", i, "title", currentTitle)

			// 点击匹配的搜索结果
			element.MustClick()

			// 获取新打开页面的内容
			pages, err := b.browser.Pages()
			if err != nil {
				return "", fmt.Errorf("failed to get pages: %w", err)
			}

			for _, p := range pages {
				if p != page {
					// 获取页面内容
					return b.getPageContent(p, title)
				}
			}

			return "", fmt.Errorf("failed to get content for search result: %s", title)
		}
	}

	return "", fmt.Errorf("no search result found with title: %s", title)
}

// getPageContent 获取页面内容
func (b *Browser) getPageContent(page *rod.Page, title string) (string, error) {
	// 等待页面加载完成
	page.MustWaitLoad()

	// 获取页面标题
	pageTitleElement, err := page.Timeout(elementTimeout).Element("h1")
	if err != nil {
		utils.Debug("Failed to find page title, using provided title")
	} else {
		pageTitle := strings.TrimSpace(pageTitleElement.MustText())
		utils.Debug("Page title", "title", pageTitle)
	}

	// 获取主要内容，尝试多个选择器
	var contentElement *rod.Element
	selectors := []string{".doc-view", "main", "article", ".content-area", ".main-content", "[class*='content']"}

	for _, selector := range selectors {
		contentElement, err = page.Timeout(elementTimeout).Element(selector)
		if err == nil {
			utils.Debug("Found content element", "selector", selector)
			break
		}
	}

	if contentElement == nil {
		return "", fmt.Errorf("failed to find content with any selector")
	}

	content := strings.TrimSpace(contentElement.MustText())

	// 关闭页面
	page.Close()

	// 返回格式化的内容
	result := fmt.Sprintf("# %s\n\n%s", title, content)
	return result, nil
}

// GetNavigation 获取导航目录
func (b *Browser) GetNavigation() ([]models.NavigationItem, error) {
	utils.Debug("Getting navigation")

	// 导航到包含导航菜单的页面
	page, err := b.NewPage("https://act.mihoyo.com/ys/ugc/tutorial/detail/mh29wpicgvh0")
	if err != nil {
		return nil, fmt.Errorf("failed to create navigation page: %w", err)
	}
	defer page.Close()

	// 等待页面加载完成
	page.MustWaitLoad()

	// 等待导航菜单容器出现，使用rod的内置等待方法
	scrollbarElement, err := page.Timeout(pageLoadTimeout).Element(".tw-scrollbar")
	if err != nil {
		return nil, fmt.Errorf("failed to find navigation scrollbar: %w", err)
	}

	// 确保元素可见并有子元素
	utils.Debug("Found scrollbar, checking for child elements")

	utils.Debug("Found navigation scrollbar")

	// 在导航容器内查找所有链接，等同于 JavaScript 的 document.querySelector('.tw-scrollbar').querySelectorAll('a')
	linkElements, err := scrollbarElement.Elements("a")
	if err != nil {
		return nil, fmt.Errorf("failed to find any links in navigation: %w", err)
	}

	utils.Debug("Found tutorial link elements", "count", len(linkElements))

	var navItems []models.NavigationItem

	for i, element := range linkElements {
		// 获取标题
		title := strings.TrimSpace(element.MustText())
		if title == "" {
			utils.Debug("Empty title, skipping", "index", i)
			continue
		}

		// 获取URL
		href, err := element.Attribute("href")
		if err != nil || href == nil {
			utils.Debug("Failed to get href for navigation item", "index", i, "title", title)
			continue
		}

		rawURL := *href

		// 验证URL包含预期的路径
		if !strings.Contains(rawURL, "/ys/ugc/tutorial/detail/") {
			utils.Debug("URL doesn't contain expected path, skipping", "index", i, "url", rawURL)
			continue
		}

		// 提取ID - 从URL中提取最后的部分
		parts := strings.Split(rawURL, "/")
		id := ""
		if len(parts) > 0 {
			id = parts[len(parts)-1]
		}

		if id == "" {
			utils.Debug("Failed to extract ID from URL", "index", i, "url", rawURL)
			continue
		}

		item := models.NavigationItem{
			Title: title,
			URL:   id, // 只返回ID，内部使用时拼接
		}

		navItems = append(navItems, item)
		utils.Debug("Added navigation item", "index", i, "title", title, "id", id)
	}

	utils.Debug("Navigation completed", "items", len(navItems))
	return navItems, nil
}

// GetTutorial 获取教程内容，接收ID并内部拼接URL
func (b *Browser) GetTutorial(id string) (*models.Tutorial, error) {
	utils.Debug("Getting tutorial", "id", id)

	// 内部拼接完整URL
	fullURL := fmt.Sprintf("https://act.mihoyo.com/ys/ugc/tutorial/detail/%s", id)

	page, err := b.NewPage(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create tutorial page: %w", err)
	}
	defer page.Close()

	// 等待页面加载完成
	page.MustWaitLoad()

	// 获取标题
	titleElement, err := page.Timeout(elementTimeout).Element("h1")
	if err != nil {
		return nil, fmt.Errorf("failed to find title: %w", err)
	}
	title := strings.TrimSpace(titleElement.MustText())

	// 获取主要内容，尝试多个选择器
	var contentElement *rod.Element
	selectors := []string{".doc-view"}

	for _, selector := range selectors {
		contentElement, err = page.Timeout(elementTimeout).Element(selector)
		if err == nil {
			utils.Debug("Found content element", "selector", selector)
			break
		}
	}

	if contentElement == nil {
		return nil, fmt.Errorf("failed to find content with any selector: %w", err)
	}

	content := strings.TrimSpace(contentElement.MustText())

	tutorial := &models.Tutorial{
		URL:         id, // 存储ID而不是完整URL
		Title:       title,
		Content:     content,
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(24 * time.Hour),
	}

	utils.Debug("Tutorial retrieved", "title", title, "content_length", len(content))
	return tutorial, nil
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

// parseNodeGraphList 使用rod原生HTML解析解析节点图列表内容
func (b *Browser) parseNodeGraphList(page *rod.Page) []models.NodeGraphItem {
	var nodeGraphs []models.NodeGraphItem

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), elementTimeout)
	defer cancel()

	// 查找所有h2标题元素（节点标题）
	h2Elements, err := page.Context(ctx).Elements("h2")
	if err != nil {
		utils.Debug("Failed to find h2 elements", "error", err)
		return nodeGraphs
	}

	utils.Debug("Found h2 elements", "count", len(h2Elements))

	for i, h2Element := range h2Elements {
		// 获取h2文本内容作为节点名称
		nodeName := strings.TrimSpace(h2Element.MustText())
		if nodeName == "" {
			utils.Debug("Empty h2 text, skipping", "index", i)
			continue
		}

		// 查找下一个同级元素，通常是包含节点描述的p元素或div
		var description strings.Builder

		// 尝试获取h2之后的下一个p元素
		// 使用CSS选择器查找下一个p元素
		nextPElements, err := h2Element.Elements("p")
		if err == nil && len(nextPElements) > 0 {
			descText := strings.TrimSpace(nextPElements[0].MustText())
			if descText != "" && !strings.Contains(descText, "节点参数") && !strings.Contains(descText, "参数类型") {
				description.WriteString(descText)
			}
		}

		// 如果没有找到描述，尝试查找包含"节点功能"的p元素及其后的描述
		if description.Len() == 0 {
			// 查找包含"节点功能"的p元素，然后获取后续的描述文本
			allElements, err := page.Elements("*")
			if err == nil {
				foundNodeFunc := false
				for _, element := range allElements {
					elementText := strings.TrimSpace(element.MustText())

					// 如果找到包含"节点功能"的元素，标记它
					if strings.Contains(elementText, "节点功能") && !strings.Contains(elementText, "节点参数") {
						foundNodeFunc = true
						continue
					}

					// 如果已经找到"节点功能"，收集后续的描述文本直到遇到"节点参数"
					if foundNodeFunc && elementText != "" &&
					   !strings.Contains(elementText, "节点参数") &&
					   !strings.Contains(elementText, "参数类型") &&
					   !strings.Contains(elementText, "节点功能") {

						// 跳过图片元素，通过检查元素内容来判断
						if len(strings.TrimSpace(element.MustText())) > 0 {
							if description.Len() > 0 {
								description.WriteString(" ")
							}
							description.WriteString(elementText)
						}
					}

					// 如果遇到"节点参数"，停止收集描述
					if foundNodeFunc && strings.Contains(elementText, "节点参数") {
						break
					}
				}
			}
		}

		// 创建节点图项
		nodeGraph := models.NodeGraphItem{
			Name:        nodeName,
			Description: strings.TrimSpace(description.String()),
		}

		nodeGraphs = append(nodeGraphs, nodeGraph)
		utils.Debug("Added node graph", "index", i, "name", nodeName, "description", nodeGraph.Description)
	}

	utils.Debug("Parsed node graphs", "total", len(nodeGraphs))
	return nodeGraphs
}

// parseNodeGraphListSimple 使用简化的解析方式，只提取节点名称
func (b *Browser) parseNodeGraphListSimple(page *rod.Page) []models.NodeGraphItem {
	var nodeGraphs []models.NodeGraphItem

	// 查找所有h2标题元素（节点标题）
	h2Elements, err := page.Elements("h2")
	if err != nil {
		utils.Debug("Failed to find h2 elements", "error", err)
		return nodeGraphs
	}

	utils.Debug("Found h2 elements", "count", len(h2Elements))

	for i, h2Element := range h2Elements {
		// 获取h2文本内容作为节点名称
		nodeName := strings.TrimSpace(h2Element.MustText())

		// 简单的过滤：只跳过空标题
		if nodeName == "" {
			utils.Debug("Empty title, skipping", "index", i)
			continue
		}

		// 只返回节点名称，不返回描述
		nodeGraph := models.NodeGraphItem{
			Name:        nodeName,
			Description: "", // 节点图列表只返回名称，详情通过另一个工具获取
		}

		nodeGraphs = append(nodeGraphs, nodeGraph)
		utils.Debug("Added node graph name", "index", i, "name", nodeName)
	}

	utils.Debug("Parsed node graph names", "total", len(nodeGraphs))
	return nodeGraphs
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

// parseAllNodeDetails 一次性解析所有节点的详细信息
func (b *Browser) parseAllNodeDetails(page *rod.Page, h2Elements []*rod.Element, clientType string, nodeType string) []*models.NodeGraphDetails {
	var allNodes []*models.NodeGraphDetails

	// 获取页面中的所有p元素，用于描述查找
	allPElements, _ := page.Elements("p")

	for i, h2Element := range h2Elements {
		// 获取h2文本内容作为节点名称
		nodeName := strings.TrimSpace(h2Element.MustText())
		utils.Debug("Processing h2 element", "index", i, "node_name", nodeName)

		// 简单的过滤：只跳过空标题
		if nodeName == "" {
			utils.Debug("Empty title, skipping", "index", i)
			continue
		}

		// 获取节点描述
		var description strings.Builder

		// 尝试获取h2之后的下一个p元素
		nextPElements, err := h2Element.Elements("p")
		if err == nil && len(nextPElements) > 0 {
			descText := strings.TrimSpace(nextPElements[0].MustText())
			if descText != "" && !strings.Contains(descText, "节点参数") && !strings.Contains(descText, "参数类型") {
				description.WriteString(descText)
			}
		}

		// 如果没有找到描述，尝试查找包含"节点功能"的p元素
		if description.Len() == 0 {
			foundNodeFunc := false
			for _, pElement := range allPElements {
				pText := strings.TrimSpace(pElement.MustText())
				if strings.Contains(pText, "节点功能") && !strings.Contains(pText, "节点参数") {
					foundNodeFunc = true
					continue
				}
				if foundNodeFunc && pText != "" &&
				   !strings.Contains(pText, "节点参数") &&
				   !strings.Contains(pText, "参数类型") {
					if description.Len() > 0 {
						description.WriteString(" ")
					}
					description.WriteString(pText)
				}
				if foundNodeFunc && strings.Contains(pText, "节点参数") {
					break
				}
			}
		}

		// 解析参数表格、输入输出、示例等
		// 对于节点列表，我们不需要详细的参数信息，所以返回空值
		var parameters, inputs, outputs []models.Param
		var example string

		nodeDetails := &models.NodeGraphDetails{
			NodeName:     nodeName,
			Description:  strings.TrimSpace(description.String()),
			ClientType:   clientType,
			NodeType:     nodeType,
			Parameters:   parameters,
			Inputs:       inputs,
			Outputs:      outputs,
			Example:      example,
			LastUpdated:  time.Now(),
		}

		allNodes = append(allNodes, nodeDetails)
		utils.Debug("Added node to list", "index", i, "name", nodeName, "description_length", len(nodeDetails.Description))
	}

	return allNodes
}

// parseSingleNodeDetails 解析单个节点的详细信息
func (b *Browser) parseSingleNodeDetails(page *rod.Page, h2Element *rod.Element, nodeName string, clientType string, nodeType string) *models.NodeGraphDetails {
	// 获取节点描述
	var description strings.Builder

	// 尝试获取h2之后的下一个p元素
	// 使用CSS选择器查找下一个p元素
	nextPElements, err := h2Element.Elements("p")
	if err == nil && len(nextPElements) > 0 {
		descText := strings.TrimSpace(nextPElements[0].MustText())
		if descText != "" && !strings.Contains(descText, "节点参数") && !strings.Contains(descText, "参数类型") {
			description.WriteString(descText)
		}
	}

	// 如果没有找到描述，尝试查找包含"节点功能"的p元素
	if description.Len() == 0 {
		pElements, err := page.Elements("p")
		if err == nil {
			foundNodeFunc := false
			for _, pElement := range pElements {
				pText := strings.TrimSpace(pElement.MustText())
				if strings.Contains(pText, "节点功能") && !strings.Contains(pText, "节点参数") {
					foundNodeFunc = true
					continue
				}
				if foundNodeFunc && pText != "" &&
				   !strings.Contains(pText, "节点参数") &&
				   !strings.Contains(pText, "参数类型") {
					if description.Len() > 0 {
						description.WriteString(" ")
					}
					description.WriteString(pText)
				}
				if foundNodeFunc && strings.Contains(pText, "节点参数") {
					break
				}
			}
		}
	}

	// 解析参数表格、输入输出、示例等
	parameters, inputs, outputs, example := b.parseNodeGraphDetails(page, nodeName)

	nodeDetails := &models.NodeGraphDetails{
		NodeName:     nodeName,
		Description:  strings.TrimSpace(description.String()),
		ClientType:   clientType,
		NodeType:     nodeType,
		Parameters:   parameters,
		Inputs:       inputs,
		Outputs:      outputs,
		Example:      example,
		LastUpdated:  time.Now(),
	}

	return nodeDetails
}

// parseNodeGraphListWithTimeout 使用rod原生HTML解析解析节点图列表内容，带超时控制
func (b *Browser) parseNodeGraphListWithTimeout(page *rod.Page, timeout time.Duration) []models.NodeGraphItem {
	var nodeGraphs []models.NodeGraphItem

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 查找所有h2标题元素（节点标题）
	h2Elements, err := page.Context(ctx).Elements("h2")
	if err != nil {
		utils.Debug("Failed to find h2 elements", "error", err)
		return nodeGraphs
	}

	utils.Debug("Found h2 elements", "count", len(h2Elements))

	for i, h2Element := range h2Elements {
		// 检查上下文是否超时
		select {
		case <-ctx.Done():
			utils.Debug("Context timeout during parsing", "parsed_count", i)
			return nodeGraphs
		default:
		}

		// 获取h2文本内容作为节点名称
		nodeName := strings.TrimSpace(h2Element.MustText())
		if nodeName == "" {
			utils.Debug("Empty h2 text, skipping", "index", i)
			continue
		}

		// 查找下一个同级元素，通常是包含节点描述的p元素或div
		var description strings.Builder

		// 尝试获取h2之后的下一个p元素
		// 使用CSS选择器查找下一个p元素
		nextPElements, err := h2Element.Elements("p")
		if err == nil && len(nextPElements) > 0 {
			descText := strings.TrimSpace(nextPElements[0].MustText())
			if descText != "" && !strings.Contains(descText, "节点参数") && !strings.Contains(descText, "参数类型") {
				description.WriteString(descText)
			}
		}

		// 如果没有找到描述，尝试查找包含"节点功能"的p元素及其后的描述
		if description.Len() == 0 {
			// 查找包含"节点功能"的p元素，然后获取后续的描述文本
			allElements, err := page.Elements("*")
			if err == nil {
				foundNodeFunc := false
				for _, element := range allElements {
					// 检查上下文是否超时
					select {
					case <-ctx.Done():
						utils.Debug("Context timeout during element parsing", "node_name", nodeName)
						break
					default:
					}

					elementText := strings.TrimSpace(element.MustText())

					// 如果找到包含"节点功能"的元素，标记它
					if strings.Contains(elementText, "节点功能") && !strings.Contains(elementText, "节点参数") {
						foundNodeFunc = true
						continue
					}

					// 如果已经找到"节点功能"，收集后续的描述文本直到遇到"节点参数"
					if foundNodeFunc && elementText != "" &&
					   !strings.Contains(elementText, "节点参数") &&
					   !strings.Contains(elementText, "参数类型") &&
					   !strings.Contains(elementText, "节点功能") {

						// 跳过图片元素，通过检查元素内容来判断
						if len(strings.TrimSpace(element.MustText())) > 0 {
							if description.Len() > 0 {
								description.WriteString(" ")
							}
							description.WriteString(elementText)
						}
					}

					// 如果遇到"节点参数"，停止收集描述
					if foundNodeFunc && strings.Contains(elementText, "节点参数") {
						break
					}
				}
			}
		}

		// 创建节点图项
		nodeGraph := models.NodeGraphItem{
			Name:        nodeName,
			Description: strings.TrimSpace(description.String()),
		}

		nodeGraphs = append(nodeGraphs, nodeGraph)
		utils.Debug("Added node graph", "index", i, "name", nodeName, "description", nodeGraph.Description)
	}

	utils.Debug("Parsed node graphs", "total", len(nodeGraphs))
	return nodeGraphs
}

// parseNodeGraphDetailsForNode 解析指定节点的详细信息，使用预获取的表格
func (b *Browser) parseNodeGraphDetailsForNode(page *rod.Page, nodeName string, allTables []*rod.Element) ([]models.Param, []models.Param, []models.Param, string) {
	var parameters, inputs, outputs []models.Param
	var example string

	// 查找包含节点名称的h2元素
	h2Elements, err := page.Elements("h2")
	if err != nil {
		utils.Debug("Failed to find h2 elements", "error", err)
		return parameters, inputs, outputs, example
	}

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

	// 查找该节点后的所有兄弟元素，直到遇到下一个h2或页面结束
	allSiblings, err := targetH2.Elements("*")
	if err != nil {
		utils.Debug("Failed to find following siblings", "node_name", nodeName, "error", err)
		return parameters, inputs, outputs, example
	}

	// 遍历兄弟元素查找表格和示例
	foundTable := false
	var exampleText strings.Builder

	for _, sibling := range allSiblings {
		siblingText := strings.TrimSpace(sibling.MustText())

		// 检查是否遇到下一个节点标题（停止解析）
		if sibling.MustMatches("h2") {
			break
		}

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

// parseNodeGraphDetails 解析节点图详细信息，包括参数表格
func (b *Browser) parseNodeGraphDetails(page *rod.Page, nodeName string) ([]models.Param, []models.Param, []models.Param, string) {
	var parameters, inputs, outputs []models.Param
	var example string

	// 查找包含节点名称的h2元素
	h2Elements, err := page.Elements("h2")
	if err != nil {
		utils.Debug("Failed to find h2 elements", "error", err)
		return parameters, inputs, outputs, example
	}

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

	// 查找该节点后的所有兄弟元素，直到遇到下一个h2或页面结束
	allSiblings, err := targetH2.Elements("*")
	if err != nil {
		utils.Debug("Failed to find following siblings", "node_name", nodeName, "error", err)
		return parameters, inputs, outputs, example
	}

	// 遍历兄弟元素查找表格和示例
	foundTable := false
	var exampleText strings.Builder

	for _, sibling := range allSiblings {
		siblingText := strings.TrimSpace(sibling.MustText())

		// 检查是否遇到下一个节点标题（停止解析）
		if sibling.MustMatches("h2") {
			break
		}

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