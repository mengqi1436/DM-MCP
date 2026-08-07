package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	registerBatchOpsTools()
}

func registerBatchOpsTools() {
	RegisterTool(ToolInfo{
		Name:        "batch_update",
		Category:    "dml",
		Description: "批量更新（多条不同WHERE条件），事务包装。参数: table(必填), updates(必填)-数组[{data:{字段:值}, where:条件}]",
		Params:      []string{"table", "updates"},
	}, handleBatchUpdate)

	RegisterTool(ToolInfo{
		Name:        "batch_delete",
		Category:    "dml",
		Description: "批量删除（多条不同WHERE条件），事务包装。参数: table(必填), wheres(必填)-WHERE条件数组",
		Params:      []string{"table", "wheres"},
	}, handleBatchDelete)

	RegisterTool(ToolInfo{
		Name:        "batch_query",
		Category:    "query",
		Description: "批量执行多条SELECT查询，返回各自结果。参数: queries(必填)-SQL数组",
		Params:      []string{"queries"},
	}, handleBatchQuery)

	RegisterTool(ToolInfo{
		Name:        "execute_sql",
		Category:    "advanced",
		Description: "执行任意SQL（自动判断SELECT/DML/DDL）。参数: sql(必填)",
		Params:      []string{"sql"},
	}, handleExecuteSQL)

	RegisterTool(ToolInfo{
		Name:        "batch_execute_sql",
		Category:    "advanced",
		Description: "批量执行多条任意SQL。参数: statements(必填)-SQL数组, atomic(可选,默认false)",
		Params:      []string{"statements", "atomic"},
	}, handleBatchExecuteSQL)

	RegisterTool(ToolInfo{
		Name:        "merge",
		Category:    "dml",
		Description: "MERGE INTO（UPSERT）。参数: table(必填), data(必填)-数据对象, match_columns(必填)-匹配列数组",
		Params:      []string{"table", "data", "match_columns"},
	}, handleMerge)

	RegisterTool(ToolInfo{
		Name:        "export_table_data",
		Category:    "import",
		Description: "导出表数据为INSERT语句或JSON。参数: table_name(必填), format(可选,insert|json,默认json), where(可选), limit(可选,默认1000)",
		Params:      []string{"table_name", "format", "where", "limit"},
	}, handleExportTableData)
}

func handleBatchUpdate(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	updates, ok := params["updates"].([]interface{})
	if !ok || len(updates) == 0 {
		return nil, fmt.Errorf("参数 updates 是必需的且不能为空")
	}

	statements := make([]string, 0, len(updates))
	for i, u := range updates {
		um, ok := u.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("updates[%d] 必须是对象", i)
		}
		data, ok := um["data"].(map[string]interface{})
		if !ok || len(data) == 0 {
			return nil, fmt.Errorf("updates[%d].data 是必需的", i)
		}
		where := getString(um, "where")
		if where == "" {
			return nil, fmt.Errorf("updates[%d].where 是必需的", i)
		}

		var setClauses []string
		for col, val := range data {
			valBytes, _ := json.Marshal(val)
			setClauses = append(setClauses, fmt.Sprintf("%s = %s", col, string(valBytes)))
		}
		statements = append(statements, fmt.Sprintf("UPDATE %s SET %s WHERE %s",
			table, strings.Join(setClauses, ", "), where))
	}

	if err := executeTransactionDB(statements); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"updated": len(statements),
	}, nil
}

func handleBatchDelete(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	wheres, err := stringSliceParam(params, "wheres")
	if err != nil {
		return nil, err
	}

	statements := make([]string, 0, len(wheres))
	for _, where := range wheres {
		statements = append(statements, fmt.Sprintf("DELETE FROM %s WHERE %s", table, where))
	}

	if err := executeTransactionDB(statements); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"deleted": len(statements),
	}, nil
}

func handleBatchQuery(params map[string]interface{}) (interface{}, error) {
	raw, ok := params["queries"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("参数 queries 是必需的且不能为空")
	}

	results := make([]map[string]interface{}, 0, len(raw))
	okCount := 0
	for i, q := range raw {
		sql, ok := q.(string)
		if !ok || sql == "" {
			results = append(results, map[string]interface{}{
				"index": i,
				"ok":    false,
				"error": "queries 中每项必须是非空字符串",
			})
			continue
		}
		rows, err := queryDB(sql)
		item := map[string]interface{}{"index": i, "sql": sql}
		if err != nil {
			item["ok"] = false
			item["error"] = err.Error()
		} else {
			item["ok"] = true
			item["rows"] = rows
			item["count"] = len(rows)
			okCount++
		}
		results = append(results, item)
	}

	return map[string]interface{}{
		"success":    okCount == len(raw),
		"total":      len(raw),
		"ok_count":   okCount,
		"fail_count": len(raw) - okCount,
		"results":    results,
	}, nil
}

func handleExecuteSQL(params map[string]interface{}) (interface{}, error) {
	sql := strings.TrimSpace(getString(params, "sql"))
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	upper := strings.ToUpper(sql)
	isSelect := strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "DESC") || strings.HasPrefix(upper, "SP_")

	if isSelect {
		results, err := queryDB(sql)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"type":  "query",
			"rows":  results,
			"count": len(results),
		}, nil
	}

	isDDL := strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP") || strings.HasPrefix(upper, "TRUNCATE") ||
		strings.HasPrefix(upper, "COMMENT") || strings.HasPrefix(upper, "GRANT") ||
		strings.HasPrefix(upper, "REVOKE")

	if isDDL {
		if err := executeDDLDB(sql); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"type":    "ddl",
			"success": true,
		}, nil
	}

	affected, err := executeDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"type":          "dml",
		"success":       true,
		"affected_rows": affected,
	}, nil
}

func handleBatchExecuteSQL(params map[string]interface{}) (interface{}, error) {
	raw, ok := params["statements"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("参数 statements 是必需的且不能为空")
	}

	statements := make([]string, 0, len(raw))
	for i, s := range raw {
		sql, ok := s.(string)
		if !ok || strings.TrimSpace(sql) == "" {
			return nil, fmt.Errorf("statements[%d] 必须是非空字符串", i)
		}
		statements = append(statements, strings.TrimSpace(sql))
	}

	atomic := getBoolOrDefault(params, "atomic", false)

	if atomic {
		if err := executeTransactionDB(statements); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success":  true,
			"atomic":   true,
			"executed": len(statements),
		}, nil
	}

	results := make([]map[string]interface{}, 0, len(statements))
	okCount := 0
	for i, stmt := range statements {
		item := map[string]interface{}{"index": i, "statement": stmt}

		upper := strings.ToUpper(stmt)
		isSelect := strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")
		if isSelect {
			rows, err := queryDB(stmt)
			if err != nil {
				item["ok"] = false
				item["error"] = err.Error()
			} else {
				item["ok"] = true
				item["count"] = len(rows)
				okCount++
			}
		} else {
			_, err := executeDB(stmt)
			if err != nil {
				item["ok"] = false
				item["error"] = err.Error()
			} else {
				item["ok"] = true
				okCount++
			}
		}
		results = append(results, item)
	}

	return map[string]interface{}{
		"success":    okCount == len(statements),
		"atomic":     false,
		"total":      len(statements),
		"ok_count":   okCount,
		"fail_count": len(statements) - okCount,
		"results":    results,
	}, nil
}

func handleMerge(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	data, ok := params["data"].(map[string]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("参数 data 是必需的且不能为空")
	}

	matchColsRaw, ok := params["match_columns"].([]interface{})
	if !ok || len(matchColsRaw) == 0 {
		return nil, fmt.Errorf("参数 match_columns 是必需的且不能为空")
	}
	matchCols := make([]string, 0, len(matchColsRaw))
	for _, v := range matchColsRaw {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("match_columns 中每项必须是非空字符串")
		}
		matchCols = append(matchCols, s)
	}

	var allCols []string
	var allVals []string
	i := 1
	for col := range data {
		allCols = append(allCols, col)
		allVals = append(allVals, fmt.Sprintf(":%d", i))
		i++
	}

	var onClauses []string
	for _, mc := range matchCols {
		onClauses = append(onClauses, fmt.Sprintf("t.%s = s.%s", mc, mc))
	}

	var updateSets []string
	for col, val := range data {
		isMatch := false
		for _, mc := range matchCols {
			if col == mc {
				isMatch = true
				break
			}
		}
		if !isMatch {
			_ = val
			updateSets = append(updateSets, fmt.Sprintf("t.%s = s.%s", col, col))
		}
	}

	srcCols := make([]string, len(allCols))
	srcSelects := make([]string, len(allCols))
	for idx, col := range allCols {
		srcCols[idx] = col
		srcSelects[idx] = fmt.Sprintf("%s AS %s", allVals[idx], col)
	}

	sql := fmt.Sprintf("MERGE INTO %s t USING (SELECT %s FROM DUAL) s ON (%s)",
		table,
		strings.Join(srcSelects, ", "),
		strings.Join(onClauses, " AND "))

	if len(updateSets) > 0 {
		sql += " WHEN MATCHED THEN UPDATE SET " + strings.Join(updateSets, ", ")
	}
	sql += fmt.Sprintf(" WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s)",
		strings.Join(allCols, ", "),
		strings.Join(allVals, ", "))

	values := make([]interface{}, 0, len(data))
	for _, val := range data {
		values = append(values, val)
	}

	affected, err := executeDB(sql, values...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":       true,
		"affected_rows": affected,
	}, nil
}

func handleExportTableData(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	format := getString(params, "format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "insert" {
		return nil, fmt.Errorf("参数 format 必须是 json 或 insert")
	}

	where := getString(params, "where")
	limit := getInt(params, "limit", 1000)
	if limit < 1 {
		limit = 1000
	}

	sql := fmt.Sprintf("SELECT * FROM %s", tableName)
	if where != "" {
		sql += " WHERE " + where
	}
	sql += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	if format == "json" {
		return map[string]interface{}{
			"table_name": tableName,
			"format":     "json",
			"count":      len(rows),
			"data":       rows,
		}, nil
	}

	var insertSQLs []string
	for _, row := range rows {
		var cols []string
		var vals []string
		for col, val := range row {
			cols = append(cols, col)
			if val == nil {
				vals = append(vals, "NULL")
			} else {
				valBytes, _ := json.Marshal(val)
				vals = append(vals, string(valBytes))
			}
		}
		insertSQLs = append(insertSQLs, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
			tableName, strings.Join(cols, ", "), strings.Join(vals, ", ")))
	}

	return map[string]interface{}{
		"table_name": tableName,
		"format":     "insert",
		"count":      len(insertSQLs),
		"statements": insertSQLs,
	}, nil
}
