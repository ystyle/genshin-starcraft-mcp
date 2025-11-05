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


