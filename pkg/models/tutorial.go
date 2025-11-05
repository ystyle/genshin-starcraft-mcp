package models

import "time"

// SearchResult 搜索结果
type SearchResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Section     string `json:"section,omitempty"`
}

// Section 教程章节
type Section struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Level    int       `json:"level"`
	Content  string    `json:"content"`
	Tables   []Table   `json:"tables,omitempty"`
	Params   []Param   `json:"params,omitempty"`
	Children []Section `json:"children,omitempty"`
}

// Table 表格
type Table struct {
	Headers []string        `json:"headers"`
	Rows    [][]string      `json:"rows"`
	Caption string          `json:"caption,omitempty"`
}

// Param 参数
type Param struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// Tutorial 教程
type Tutorial struct {
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Summary      string     `json:"summary"`
	Outline      []Section  `json:"outline"`
	Sections     []Section  `json:"sections"`
	LastUpdated  time.Time  `json:"last_updated"`
	CacheExpiry  time.Time  `json:"cache_expiry"`
}

// CachedPage 缓存的页面
type CachedPage struct {
	Tutorial    *Tutorial `json:"tutorial"`
	LastUpdated time.Time  `json:"last_updated"`
	Expires     time.Time  `json:"expires"`
}

// NavigationItem 导航项
type NavigationItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// NodeGraphItem 节点图项
type NodeGraphItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"` // 节点所属分类
}

// NodeGraphDetails 节点图详细信息
type NodeGraphDetails struct {
	NodeName     string   `json:"node_name"`
	Description  string   `json:"description"`
	ClientType   string   `json:"client_type,omitempty"`   // 客户端类型：server 或 client
	NodeType     string   `json:"node_type,omitempty"`     // 节点类型：general, query, operation 等
	Parameters   []Param  `json:"parameters,omitempty"`
	Inputs       []Param  `json:"inputs,omitempty"`
	Outputs      []Param  `json:"outputs,omitempty"`
	Example      string   `json:"example,omitempty"`
	LastUpdated  time.Time `json:"last_updated"`
}

// NodeGraphPage 缓存完整的节点图页面解析结果
type NodeGraphPage struct {
	ClientType string                        `json:"client_type"`
	NodeType   string                        `json:"node_type"`
	Nodes      []*NodeGraphDetails          `json:"nodes"` // 按顺序排列的节点列表
	LastUpdated time.Time                    `json:"last_updated"`
}