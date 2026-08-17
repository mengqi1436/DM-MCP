package tools

import (
	"dm-mcp2/config"
	"fmt"
	"strings"
)

func init() {
	registerQueryTools()
}

func registerQueryTools() {
	RegisterTool(ToolInfo{
		Name:        "query",
		Category:    "query",
		Description: "执行SQL SELECT查询语句，返回查询结果。参数: sql(必填), limit(可选,默认1000)",
		Params:      []string{"sql", "limit"},
	}, handleQuery)

	RegisterTool(ToolInfo{
		Name:        "query_one",
		Category:    "query",
		Description: "执行SQL查询，只返回第一条记录。参数: sql(必填)",
		Params:      []string{"sql"},
	}, handleQueryOne)

	RegisterTool(ToolInfo{
		Name:        "query_paginated",
		Category:    "query",
		Description: "执行分页查询。参数: sql, page(默认1), page_size(默认20)",
		Params:      []string{"sql", "page", "page_size"},
	}, handleQueryPaginated)

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

	sql = applyQueryLimit(sql, params)

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"rows":  results,
		"count": len(results),
	}, nil
}

// applyQueryLimit 若 SQL 未显式指定 LIMIT，则追加默认行数上限（DM_QUERY_LIMIT，
// 可通过参数 limit 覆盖），防止无界结果集耗尽内存。
func applyQueryLimit(sql string, params map[string]interface{}) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.Contains(upper, " LIMIT ") || strings.HasSuffix(upper, " LIMIT") {
		return sql
	}
	limit := getInt(params, "limit", 0)
	if limit <= 0 {
		limit = config.LoadConfig().QueryLimit
	}
	if limit <= 0 {
		return sql
	}
	trimmed := strings.TrimSuffix(strings.TrimSpace(sql), ";")
	return fmt.Sprintf("%s LIMIT %d", trimmed, limit)
}

func handleQueryOne(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
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

	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d",
		strings.TrimSuffix(strings.TrimSpace(sql), ";"),
		pageSize, offset)

	results, err := queryDB(paginatedSQL)
	if err != nil {
		return nil, err
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

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		// 达梦驱动默认返回大写列名，大小写不敏感查找 cnt
		for k, v := range results[0] {
			if strings.EqualFold(k, "cnt") {
				return map[string]interface{}{
					"table": table,
					"count": v,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("统计失败")
}
