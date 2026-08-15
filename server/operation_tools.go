package server

import (
	"context"
	"dm-mcp/tools"
	"encoding/json"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newMCPServer(name, version string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, nil)
}

// registerTools 注册 control 工具与 v1 registry 中的全部操作工具。
func registerTools(s *mcp.Server) {
	registerControlTools(s)
	for _, info := range tools.GetAllTools("") {
		ti := info
		s.AddTool(buildTool(ti), handleTool(ti.Name))
	}
}

// ---------- control 工具 ----------

// registerControlTools 注册工具发现与统一执行入口（沿袭 v1 handlers.go）。
func registerControlTools(s *mcp.Server) {
	s.AddTool(&mcp.Tool{
		Name:        "dm_list_tools",
		Description: "列出所有可用的达梦数据库操作工具。可按类别筛选：query/dml/ddl/metadata/advanced/admin/import/instance",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"category": map[string]any{
					"type":        "string",
					"description": "筛选类别（可选）: query, dml, ddl, metadata, advanced, admin, import, instance",
				},
			},
		},
	}, handleListTools)

	s.AddTool(&mcp.Tool{
		Name:        "dm_execute",
		Description: "执行指定的达梦数据库操作工具。参数: tool_name(必填)-要执行的工具名称, params(可选)-工具参数(JSON对象)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tool_name": map[string]any{"type": "string", "description": "要执行的工具名称"},
				"params":    map[string]any{"type": "object", "description": "工具参数（JSON对象）"},
			},
			"required": []string{"tool_name"},
		},
	}, handleExecuteTool)
}

func handleListTools(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := parseArguments(req)
	category := ""
	if v, ok := params["category"].(string); ok {
		category = v
	}
	infos := tools.GetAllTools(category)
	return jsonResult(map[string]interface{}{
		"total":      len(infos),
		"category":   category,
		"categories": tools.GetCategories(),
		"tools":      infos,
	})
}

func handleExecuteTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := parseArguments(req)
	toolName, _ := params["tool_name"].(string)
	if toolName == "" {
		return errResult("参数 tool_name 是必需的"), nil
	}
	subParams := map[string]interface{}{}
	if m, ok := params["params"].(map[string]interface{}); ok {
		subParams = m
	}
	result, err := tools.ExecuteTool(toolName, subParams)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(result)
}

// ---------- 操作工具 ----------

// buildTool 从 v1 工具元数据构造官方 SDK 的 Tool（含 JSON Schema InputSchema）。
func buildTool(info tools.ToolInfo) *mcp.Tool {
	return &mcp.Tool{
		Name:        info.Name,
		Description: info.Description,
		InputSchema: buildInputSchema(info),
	}
}

// buildInputSchema 生成 JSON Schema 2020-12 格式的输入结构。
// 参数类型按 v1 约定推断；"必填"参数从描述中的 "(必填)" 标记提取。
var requiredParamPattern = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\(必填\)`)

func buildInputSchema(info tools.ToolInfo) map[string]any {
	properties := make(map[string]any, len(info.Params))
	seen := make(map[string]bool, len(info.Params))
	for _, p := range info.Params {
		if seen[p] {
			continue
		}
		seen[p] = true
		properties[p] = map[string]any{
			"type":        paramJSONType(p, info.Name),
			"description": "参数 " + p,
		}
	}

	var required []string
	for _, m := range requiredParamPattern.FindAllStringSubmatch(info.Description, -1) {
		if seen[m[1]] {
			required = append(required, m[1])
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// paramJSONType 按参数名推断 JSON Schema 类型（与 v1 optionForOperationParam 规则一致）。
func paramJSONType(name, toolName string) string {
	if name == "rows" && toolName == "insert_batch" {
		return "array"
	}
	if name == "params" && (toolName == "call_procedure" || toolName == "call_function" || toolName == "query" || toolName == "query_one") {
		return "array"
	}
	switch name {
	case "atomic", "cascade", "confirm", "direct", "full", "if_exists", "or_replace", "stop_service", "unique":
		return "boolean"
	case "batch_size", "cache_size", "errors", "extent_size", "increment_by", "index_option", "limit", "log_size", "max_parallel", "max_value", "page", "page_size", "port", "port_num", "rows", "size", "skip", "start_with", "timeout_seconds":
		return "number"
	case "data":
		return "object"
	case "columns", "column_types", "exclude_tables", "extra_args", "files", "indexes", "index_names", "match_columns", "params", "queries", "statements", "table_names", "tables", "updates", "wheres":
		return "array"
	default:
		return "string"
	}
}

// handleTool 适配 v1 的 ToolHandler(params) → (result, error) 到官方 SDK 的 ToolHandler。
func handleTool(name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := tools.ExecuteTool(name, parseArguments(req))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return jsonResult(result)
	}
}

// ---------- 辅助 ----------

// parseArguments 将官方 SDK 的 RawMessage 参数解析为 map。
func parseArguments(req *mcp.CallToolRequest) map[string]interface{} {
	params := map[string]interface{}{}
	if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(req.Params.Arguments, &m); err == nil && m != nil {
			params = m
		}
	}
	return params
}

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	// 使用单行 JSON：stdio 传输为 newline-delimited，多行缩进会破坏分帧。
	data, err := json.Marshal(v)
	if err != nil {
		return errResult("结果序列化失败: " + err.Error()), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
