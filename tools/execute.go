package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	registerExecuteTools()
}

func registerExecuteTools() {
	RegisterTool(ToolInfo{
		Name:        "insert",
		Category:    "dml",
		Description: "插入一条数据。参数: table(必填)-表名, data(必填)-数据对象{字段名:值}",
		Params:      []string{"table", "data"},
	}, handleInsert)

	RegisterTool(ToolInfo{
		Name:        "insert_batch",
		Category:    "dml",
		Description: "批量插入多条数据。参数: table(必填)-表名, rows(必填)-数据数组",
		Params:      []string{"table", "rows"},
	}, handleInsertBatch)

	RegisterTool(ToolInfo{
		Name:        "update",
		Category:    "dml",
		Description: "更新数据。参数: table(必填)-表名, data(必填)-更新数据, where(必填)-WHERE条件",
		Params:      []string{"table", "data", "where"},
	}, handleUpdate)

	RegisterTool(ToolInfo{
		Name:        "delete",
		Category:    "dml",
		Description: "删除数据。参数: table(必填)-表名, where(必填)-WHERE条件(防止误删)",
		Params:      []string{"table", "where"},
	}, handleDelete)
}

func handleInsert(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	data, ok := params["data"].(map[string]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("参数 data 是必需的且不能为空")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf(":%d", i))
		values = append(values, val)
		i++
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	affected, err := executeDB(sql, values...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":       true,
		"affected_rows": affected,
	}, nil
}

func handleInsertBatch(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	rows, ok := params["rows"].([]interface{})
	if !ok || len(rows) == 0 {
		return nil, fmt.Errorf("参数 rows 是必需的且不能为空")
	}

	statements := make([]string, len(rows))
	for i, row := range rows {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rows[%d] 必须是对象", i)
		}

		columns := make([]string, 0, len(rowMap))
		valuesStr := make([]string, 0, len(rowMap))

		for col, val := range rowMap {
			columns = append(columns, col)
			valBytes, _ := json.Marshal(val)
			valuesStr = append(valuesStr, string(valBytes))
		}

		statements[i] = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columns, ", "),
			strings.Join(valuesStr, ", "))
	}

	err := executeTransactionDB(statements)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":  true,
		"inserted": len(rows),
	}, nil
}

func handleUpdate(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	data, ok := params["data"].(map[string]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("参数 data 是必需的且不能为空")
	}

	where := getString(params, "where")
	if where == "" {
		return nil, fmt.Errorf("参数 where 是必需的")
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = :%d", col, i))
		values = append(values, val)
		i++
	}

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		where)

	affected, err := executeDB(sql, values...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":       true,
		"affected_rows": affected,
	}, nil
}

func handleDelete(params map[string]interface{}) (interface{}, error) {
	table := getString(params, "table")
	if table == "" {
		return nil, fmt.Errorf("参数 table 是必需的")
	}

	where := getString(params, "where")
	if where == "" {
		return nil, fmt.Errorf("参数 where 是必需的（防止误删全表数据）")
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	affected, err := executeDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":       true,
		"affected_rows": affected,
	}, nil
}
