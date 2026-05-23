package main

import (
	"context"
	"dm-mcp/tools"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleListTools 处理列出工具请求
func handleListTools(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := ""

	// 类型断言获取Arguments
	if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
		if c, ok := args["category"].(string); ok {
			category = c
		}
	}

	toolList := tools.GetAllTools(category)
	categories := tools.GetCategories()

	response := map[string]interface{}{
		"total":      len(toolList),
		"categories": categories,
		"tools":      toolList,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("序列化失败: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleExecute 处理执行工具请求
func handleExecute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 类型断言获取Arguments
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	toolName, ok := args["tool_name"].(string)
	if !ok || toolName == "" {
		return mcp.NewToolResultError("参数 tool_name 是必需的"), nil
	}

	// 获取工具参数
	var params map[string]interface{}
	if p, ok := args["params"].(map[string]interface{}); ok {
		params = p
	} else {
		params = make(map[string]interface{})
	}

	// 执行工具
	result, err := tools.ExecuteTool(toolName, params)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 序列化结果
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("结果序列化失败: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
