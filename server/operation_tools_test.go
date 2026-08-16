package server

import (
	"context"
	"dm-mcp2/tools"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newCallReq(name string, args map[string]interface{}) *mcp.CallToolRequest {
	req := &mcp.CallToolRequest{}
	req.Params = &mcp.CallToolParamsRaw{Name: name}
	if args != nil {
		raw, _ := json.Marshal(args)
		req.Params.Arguments = raw
	}
	return req
}

func TestBuildInputSchemaTypes(t *testing.T) {
	info := tools.ToolInfo{
		Name:        "query_paginated",
		Description: "执行分页查询。参数: sql(必填), page(默认1), page_size(默认20)",
		Params:      []string{"sql", "page", "page_size"},
	}
	schema := buildInputSchema(info)

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties 缺失")
	}
	if typ := props["sql"].(map[string]any)["type"]; typ != "string" {
		t.Errorf("sql 类型应为 string, got %v", typ)
	}
	if typ := props["page"].(map[string]any)["type"]; typ != "number" {
		t.Errorf("page 类型应为 number, got %v", typ)
	}

	req, ok := schema["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "sql" {
		t.Errorf("required 应包含 sql, got %v", schema["required"])
	}
}

func TestBuildInputSchemaArrayParams(t *testing.T) {
	info := tools.ToolInfo{
		Name:        "insert_batch",
		Description: "批量插入多条数据。参数: table(必填)-表名, rows(必填)-数据数组",
		Params:      []string{"table", "rows"},
	}
	schema := buildInputSchema(info)
	props := schema["properties"].(map[string]any)
	if typ := props["rows"].(map[string]any)["type"]; typ != "array" {
		t.Errorf("insert_batch.rows 类型应为 array, got %v", typ)
	}
	req := schema["required"].([]string)
	if len(req) != 2 {
		t.Errorf("required 应包含 table 和 rows, got %v", req)
	}
}

func TestBuildInputSchemaBooleanParams(t *testing.T) {
	info := tools.ToolInfo{
		Name:        "batch_execute_sql",
		Description: "批量执行多条任意SQL。参数: statements(必填)-SQL数组, atomic(可选,默认false)",
		Params:      []string{"statements", "atomic"},
	}
	schema := buildInputSchema(info)
	props := schema["properties"].(map[string]any)
	if typ := props["atomic"].(map[string]any)["type"]; typ != "boolean" {
		t.Errorf("atomic 类型应为 boolean, got %v", typ)
	}
	if typ := props["statements"].(map[string]any)["type"]; typ != "array" {
		t.Errorf("statements 类型应为 array, got %v", typ)
	}
}

// TestHandleListTools 验证无需数据库的 control 工具（参数解析 + JSON 输出）。
func TestHandleListTools(t *testing.T) {
	req := newCallReq("dm_list_tools", map[string]interface{}{"category": "query"})
	result, err := handleListTools(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListTools 错误: %v", err)
	}
	if result.IsError {
		t.Fatalf("不应为错误结果: %s", result.Content[0].(*mcp.TextContent).Text)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"total"`) {
		t.Errorf("结果应包含 total: %s", text)
	}
	if !strings.Contains(text, `"purpose"`) {
		t.Errorf("精简目录应包含 purpose 字段: %s", text)
	}
	// 双通道：StructuredContent 应与 Content 一致
	if result.StructuredContent == nil {
		t.Error("应返回 StructuredContent 结构化通道")
	}
	sc, _ := json.Marshal(result.StructuredContent)
	if string(sc) != text {
		t.Errorf("StructuredContent 应与 Content 一致\nContent: %s\nStructured: %s", text, string(sc))
	}
}

// TestHandleListToolsDetail 验证 detail=1 返回完整元数据（含 params）。
func TestHandleListToolsDetail(t *testing.T) {
	req := newCallReq("dm_list_tools", map[string]interface{}{"category": "query", "detail": true})
	result, err := handleListTools(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListTools 错误: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"params"`) {
		t.Errorf("detail 模式应包含 params 字段: %s", text)
	}
	if strings.Contains(text, `"purpose"`) {
		t.Errorf("detail 模式不应包含精简 purpose 字段: %s", text)
	}
}

// TestHandleGetTool 验证按名加载单个工具完整定义。
func TestHandleGetTool(t *testing.T) {
	req := newCallReq("dm_get_tool", map[string]interface{}{"tool_name": "count"})
	result, err := handleGetTool(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetTool 错误: %v", err)
	}
	if result.IsError {
		t.Fatalf("不应为错误结果: %s", result.Content[0].(*mcp.TextContent).Text)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"count"`) || !strings.Contains(text, `"params"`) {
		t.Errorf("应返回工具完整定义: %s", text)
	}
}

// TestHandleGetToolMissing 验证未知工具名返回错误。
func TestHandleGetToolMissing(t *testing.T) {
	req := newCallReq("dm_get_tool", map[string]interface{}{"tool_name": "no_such_tool"})
	result, err := handleGetTool(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetTool 错误: %v", err)
	}
	if !result.IsError {
		t.Error("未知工具名应返回错误结果")
	}
}

// TestSummarizeResultAppliedViaHandler 验证 jsonResult 对列表工具做摘要化。
func TestSummarizeResultAppliedViaHandler(t *testing.T) {
	items := make([]interface{}, 0, 100)
	for i := 0; i < 100; i++ {
		items = append(items, map[string]interface{}{"ID": i})
	}
	body, _ := tools.SummarizeResult("query", map[string]interface{}{"rows": items, "count": 100}, 20)
	result, err := jsonResult("query", body)
	if err != nil {
		t.Fatalf("jsonResult 错误: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, `"was_truncated":true`) {
		t.Errorf("截断结果应标记 was_truncated: %s", text)
	}
	if !strings.Contains(text, `"summary"`) {
		t.Errorf("截断结果应包含 summary: %s", text)
	}
}

// TestHandleToolArgumentsDecoding 验证 Arguments(RawMessage) → map 解析。
func TestHandleToolArgumentsDecoding(t *testing.T) {
	req := newCallReq("dm_list_tools", map[string]interface{}{"category": "query"})
	params := parseArguments(req)
	if params["category"] != "query" {
		t.Errorf("参数解析失败: %v", params)
	}
}

// TestHandleExecuteTool 验证 dm_execute 参数校验（缺 tool_name 时报错）。
func TestHandleExecuteTool(t *testing.T) {
	req := newCallReq("dm_execute", map[string]interface{}{})
	result, err := handleExecuteTool(context.Background(), req)
	if err != nil {
		t.Fatalf("handleExecuteTool 错误: %v", err)
	}
	if !result.IsError {
		t.Errorf("缺少 tool_name 应返回错误结果")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "tool_name") {
		t.Errorf("错误信息应提及 tool_name: %s", text)
	}
}

func TestBuildTool(t *testing.T) {
	info := tools.ToolInfo{
		Name:        "count",
		Description: "统计表记录数。参数: table(必填), where(可选)",
		Params:      []string{"table", "where"},
	}
	tool := buildTool(info)
	if tool.Name != "count" {
		t.Errorf("tool 名称错误: %s", tool.Name)
	}
	if tool.InputSchema == nil {
		t.Error("InputSchema 不应为 nil")
	}
}
