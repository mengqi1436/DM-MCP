package tools

import (
	"fmt"
	"sync"
)

// ToolInfo 工具信息
type ToolInfo struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
}

// ToolHandler 工具处理函数
type ToolHandler func(params map[string]interface{}) (interface{}, error)

var (
	toolRegistry = make(map[string]ToolInfo)
	toolHandlers = make(map[string]ToolHandler)
	mu           sync.RWMutex
)

// RegisterTool 注册工具
func RegisterTool(info ToolInfo, handler ToolHandler) {
	mu.Lock()
	defer mu.Unlock()
	toolRegistry[info.Name] = info
	toolHandlers[info.Name] = handler
}

// GetAllTools 获取所有工具（可按类别筛选）
func GetAllTools(category string) []ToolInfo {
	mu.RLock()
	defer mu.RUnlock()

	var result []ToolInfo
	for _, info := range toolRegistry {
		if category == "" || info.Category == category {
			result = append(result, info)
		}
	}
	return result
}

// GetToolInfo 获取单个工具信息
func GetToolInfo(name string) (ToolInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	info, exists := toolRegistry[name]
	return info, exists
}

// ExecuteTool 执行指定工具
func ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
	mu.RLock()
	handler, exists := toolHandlers[name]
	mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("工具 '%s' 不存在", name)
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	return handler(params)
}

// GetCategories 获取所有类别
func GetCategories() []string {
	mu.RLock()
	defer mu.RUnlock()

	categoryMap := make(map[string]bool)
	for _, info := range toolRegistry {
		categoryMap[info.Category] = true
	}

	var categories []string
	for cat := range categoryMap {
		categories = append(categories, cat)
	}
	return categories
}
