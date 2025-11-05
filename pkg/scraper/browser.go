package scraper

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"genshin-starcraft-mcp/pkg/models"
)

// Browser 浏览器实例
type Browser struct {
	browser        *rod.Browser
	nodeGraphCache map[string]*models.NodeGraphPage // 缓存完整的页面解析结构，key为clientType_nodeType
}

// NewBrowser 创建新的浏览器实例
func NewBrowser() (*Browser, error) {
	browser := rod.New().
		MustConnect().
		MustIgnoreCertErrors(true)

	return &Browser{
		browser:        browser,
		nodeGraphCache: make(map[string]*models.NodeGraphPage),
	}, nil
}

// Close 关闭浏览器
func (b *Browser) Close() error {
	if b.browser != nil {
		return b.browser.Close()
	}
	return nil
}

// NewPage 创建新页面
func (b *Browser) NewPage(url string) (*rod.Page, error) {
	page := b.browser.MustPage()

	// 导航到URL
	err := page.Navigate(url)
	if err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// 等待网络空闲
	page.MustWaitIdle()

	return page, nil
}

// WaitForElement 等待元素出现
func (b *Browser) WaitForElement(page *rod.Page, selector string, timeout time.Duration) (*rod.Element, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	element, err := page.Context(ctx).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element %s not found: %w", selector, err)
	}
	return element, nil
}

// WaitForDialog 等待对话框出现
func (b *Browser) WaitForDialog(page *rod.Page, timeout time.Duration) (*rod.Element, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	element, err := page.Context(ctx).Element("div[role=dialog]")
	if err != nil {
		return nil, fmt.Errorf("dialog not found: %w", err)
	}
	return element, nil
}