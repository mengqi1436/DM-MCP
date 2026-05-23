package tools

import (
	"dm-mcp/database"
	"fmt"
	"strings"
)

func init() {
	registerQueryTools()
}

func registerQueryTools() {
	// query - 执行SQL查询
	RegisterTool(ToolInfo{
		Name:        "query",
		Category:    "query",
		Description: "执行SQL SELECT查询语句，返回查询结果",
		Params:      []string{"sql", "params"},
	}, handleQuery)

	// query_one - 查询单条记录
	RegisterTool(ToolInfo{
		Name:        "query_one",
		Category:    "query",
		Description: "执行SQL查询，只返回第一条记录",
		Params:      []string{"sql", "params"},
	}, handleQueryOne)

	// query_paginated - 分页查询
	RegisterTool(ToolInfo{
		Name:        "query_paginated",
		Category:    "query",
		Description: "执行分页查询。参数: sql, page(默认1), page_size(默认20)",
		Params:      []string{"sql", "page", "page_size"},
	}, handleQueryPaginated)

	// count - 统计记录数
	RegisterTool(ToolInfo{
		Name:        "count",
		Category:    "query",
		Description: "统计表记录数。参数: table(必填), where(可选)",
		Params:      []string{"table", "where"},
	}, handleCount)
}

func handleQuery(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	results, err := database.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	return map[string]interface{}{
		"rows":  results,
		"count": len(results),
	}, nil
}

func handleQueryOne(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	results, err := database.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	if len(results) == 0 {
		return map[string]interface{}{
			"found": false,
			"row":   nil,
		}, nil
	}

	return map[string]interface{}{
		"found": true,
		"row":   results[0],
	}, nil
}

func handleQueryPaginated(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	page := getInt(params, "page", 1)
	pageSize := getInt(params, "page_size", 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 达梦数据库分页语法
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d",
		strings.TrimSuffix(strings.TrimSpace(sql), ";"),
		pageSize, offset)

	results, err := database.Query(paginatedSQL)
	if err != nil {
		return nil, fmt.Errorf("分页查询失败: %v", err)
	}

	return map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
		"count":     len(results),
		"rows":      results,
	}, nil
}

func handleCount(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	where := getString(params, "where")

	sql := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", table)
	if where != "" {
		sql += " WHERE " + where
	}

	results, err := database.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("统计失败: %v", err)
	}

	if len(results) > 0 {
		return map[string]interface{}{
			"table": table,
			"count": results[0]["cnt"],
		}, nil
	}

	return nil, fmt.Errorf("统计失败")
}
