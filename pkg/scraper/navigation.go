package scraper

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"genshin-starcraft-mcp/pkg/models"
	"genshin-starcraft-mcp/pkg/utils"
)

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