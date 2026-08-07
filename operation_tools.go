package main

import (
	"context"
	"dm-mcp/tools"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerOperationTools(s *server.MCPServer) {
	for _, info := range tools.GetAllTools("") {
		toolInfo := info
		s.AddTool(buildOperationTool(toolInfo), handleOperationTool(toolInfo.Name))
	}
}

func buildOperationTool(info tools.ToolInfo) mcp.Tool {
	options := []mcp.ToolOption{mcp.WithDescription(info.Description)}
	for _, param := range info.Params {
		options = append(options, optionForOperationParam(info.Name, param))
	}
	return mcp.NewTool(info.Name, options...)
}

func optionForOperationParam(toolName, param string) mcp.ToolOption {
	if param == "rows" && toolName == "insert_batch" {
		return mcp.WithArray(param)
	}
	if param == "params" && (toolName == "call_procedure" || toolName == "call_function" || toolName == "query" || toolName == "query_one") {
		return mcp.WithArray(param)
	}

	switch param {
	case "atomic", "cascade", "confirm", "direct", "full", "if_exists", "or_replace", "stop_service", "unique":
		return mcp.WithBoolean(param)
	case "batch_size", "cache_size", "charset", "errors", "extent_size", "increment_by", "index_option", "limit", "log_size", "max_parallel", "max_value", "page", "page_size", "port", "port_num", "rows", "size", "skip", "start_with", "timeout_seconds":
		return mcp.WithNumber(param)
	case "data":
		return mcp.WithObject(param)
	case "columns", "column_types", "exclude_tables", "extra_args", "files", "indexes", "index_names", "match_columns", "params", "queries", "statements", "table_names", "tables", "updates", "wheres":
		return mcp.WithArray(param)
	default:
		return mcp.WithString(param)
	}
}

func handleOperationTool(toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params, ok := req.Params.Arguments.(map[string]interface{})
		if !ok || params == nil {
			params = make(map[string]interface{})
		}

		result, err := tools.ExecuteTool(toolName, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("结果序列化失败: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
