package tools

import (
	"fmt"
	"strings"
)

const ddlAtomicNote = "达梦 DM 中 DDL 可能触发隐式提交（官方手册：一致性 / DM 事务相关语句），atomic=true 时仍可能无法达到直觉上的单事务全回滚；批量 DDL 推荐 atomic=false 逐条执行并检查 results。"

func init() {
	registerDDLTools()
}

func registerDDLTools() {
	RegisterTool(ToolInfo{
		Name:        "create_table",
		Category:    "ddl",
		Description: "创建表。参数: table_name-表名, columns-列定义数组[{name,type,length,primary_key,not_null,default}]",
		Params:      []string{"table_name", "columns"},
	}, handleCreateTable)

	RegisterTool(ToolInfo{
		Name:        "alter_table",
		Category:    "ddl",
		Description: "修改表结构。参数: table_name, operation(ADD/MODIFY/DROP), column, type(ADD/MODIFY时需要)",
		Params:      []string{"table_name", "operation", "column", "type"},
	}, handleAlterTable)

	RegisterTool(ToolInfo{
		Name:        "drop_table",
		Category:    "ddl",
		Description: "删除表。参数: table_name, if_exists(可选)",
		Params:      []string{"table_name", "if_exists"},
	}, handleDropTable)

	RegisterTool(ToolInfo{
		Name:        "create_index",
		Category:    "ddl",
		Description: "创建索引。参数: index_name, table_name, columns(列名数组), unique(可选)",
		Params:      []string{"index_name", "table_name", "columns", "unique"},
	}, handleCreateIndex)

	RegisterTool(ToolInfo{
		Name:        "drop_index",
		Category:    "ddl",
		Description: "删除索引。参数: index_name(可为 SCHEMA.INDEX 全名), schema(可选-与 index_name 组合), if_exists(可选)",
		Params:      []string{"index_name", "schema", "if_exists"},
	}, handleDropIndex)

	RegisterTool(ToolInfo{
		Name:        "execute_ddl",
		Category:    "ddl",
		Description: "执行DDL语句(CREATE/ALTER/DROP)。参数: sql",
		Params:      []string{"sql"},
	}, handleExecuteDDL)

	RegisterTool(ToolInfo{
		Name:        "batch_create_tables",
		Category:    "ddl",
		Description: "批量创建表（结构化）。参数: tables-数组[{table_name,columns}], atomic(可选,默认false)。" + ddlAtomicNote,
		Params:      []string{"tables", "atomic"},
	}, handleBatchCreateTables)

	RegisterTool(ToolInfo{
		Name:        "batch_create_indexes",
		Category:    "ddl",
		Description: "批量创建索引。参数: indexes-数组[{index_name,table_name,columns,unique}], atomic(可选,默认false)。" + ddlAtomicNote,
		Params:      []string{"indexes", "atomic"},
	}, handleBatchCreateIndexes)

	RegisterTool(ToolInfo{
		Name:        "batch_drop_tables",
		Category:    "ddl",
		Description: "批量删除表。参数: table_names-表名数组, if_exists(可选), atomic(可选,默认false)。" + ddlAtomicNote,
		Params:      []string{"table_names", "if_exists", "atomic"},
	}, handleBatchDropTables)

	RegisterTool(ToolInfo{
		Name:        "batch_drop_indexes",
		Category:    "ddl",
		Description: "批量删除索引。参数: index_names-名称数组(可为 SCHEMA.INDEX), if_exists(可选), atomic(可选,默认false)。" + ddlAtomicNote,
		Params:      []string{"index_names", "if_exists", "atomic"},
	}, handleBatchDropIndexes)
}

func buildCreateTableSQL(tableName string, columns []interface{}) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("table_name 不能为空")
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("columns 不能为空")
	}

	var columnDefs []string
	var primaryKeys []string

	for _, col := range columns {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			continue
		}

		name := getString(colMap, "name")
		colType := getString(colMap, "type")
		if name == "" || colType == "" {
			continue
		}

		def := name + " " + colType
		if length := getInt(colMap, "length", 0); length > 0 {
			def += fmt.Sprintf("(%d)", length)
		}
		if getBool(colMap, "not_null") {
			def += " NOT NULL"
		}
		if defaultVal := getString(colMap, "default"); defaultVal != "" {
			def += " DEFAULT " + defaultVal
		}
		if getBool(colMap, "primary_key") {
			primaryKeys = append(primaryKeys, name)
		}
		columnDefs = append(columnDefs, def)
	}

	if len(columnDefs) == 0 {
		return "", fmt.Errorf("columns 中无有效列定义")
	}

	if len(primaryKeys) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(columnDefs, ",\n  "))
	return sql, nil
}

func buildCreateIndexSQL(indexName, tableName string, columns []interface{}, unique bool) (string, error) {
	if indexName == "" || tableName == "" {
		return "", fmt.Errorf("index_name 与 table_name 不能为空")
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("columns 不能为空")
	}

	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = fmt.Sprintf("%v", col)
	}

	sql := "CREATE "
	if unique {
		sql += "UNIQUE "
	}
	sql += fmt.Sprintf("INDEX %s ON %s (%s)", indexName, tableName, strings.Join(columnNames, ", "))
	return sql, nil
}

func buildDropIndexSQL(indexName, schema string, ifExists bool) string {
	full := strings.TrimSpace(indexName)
	if schema != "" && !strings.Contains(full, ".") {
		full = schema + "." + full
	}
	sql := "DROP INDEX "
	if ifExists {
		sql += "IF EXISTS "
	}
	sql += full
	return sql
}

func buildDropTableSQL(tableName string, ifExists bool) string {
	sql := "DROP TABLE "
	if ifExists {
		sql += "IF EXISTS "
	}
	sql += tableName
	return sql
}

func runBatchDDL(statements []string, atomic bool) (map[string]interface{}, error) {
	if len(statements) == 0 {
		return nil, fmt.Errorf("无待执行 DDL 语句")
	}

	if atomic {
		if err := executeTransactionDB(statements); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success":  true,
			"atomic":   true,
			"count":    len(statements),
			"note":     ddlAtomicNote,
			"executed": len(statements),
		}, nil
	}

	results := make([]map[string]interface{}, 0, len(statements))
	okCount := 0
	for i, stmt := range statements {
		err := executeDDLDB(stmt)
		item := map[string]interface{}{
			"index":     i,
			"statement": stmt,
			"ok":        err == nil,
		}
		if err != nil {
			item["error"] = err.Error()
		} else {
			okCount++
		}
		results = append(results, item)
	}

	return map[string]interface{}{
		"success":    okCount == len(statements),
		"atomic":     false,
		"count":      len(statements),
		"ok_count":   okCount,
		"fail_count": len(statements) - okCount,
		"results":    results,
		"note":       ddlAtomicNote,
	}, nil
}

func handleCreateTable(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	columns, ok := params["columns"].([]interface{})
	if !ok || len(columns) == 0 {
		return nil, fmt.Errorf("参数 columns 是必需的且不能为空")
	}

	sql, err := buildCreateTableSQL(tableName, columns)
	if err != nil {
		return nil, err
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("表 %s 创建成功", tableName),
	}, nil
}

func handleAlterTable(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	operation := getString(params, "operation")
	column := getString(params, "column")
	colType := getString(params, "type")

	var sql string
	switch operation {
	case "ADD":
		if colType == "" {
			return nil, fmt.Errorf("ADD操作需要指定列类型")
		}
		sql = fmt.Sprintf("ALTER TABLE %s ADD %s %s", tableName, column, colType)
	case "MODIFY":
		if colType == "" {
			return nil, fmt.Errorf("MODIFY操作需要指定列类型")
		}
		sql = fmt.Sprintf("ALTER TABLE %s MODIFY %s %s", tableName, column, colType)
	case "DROP":
		sql = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, column)
	default:
		return nil, fmt.Errorf("不支持的操作类型，请使用ADD、MODIFY或DROP")
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("表 %s 修改成功", tableName),
	}, nil
}

func handleDropTable(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	sql := buildDropTableSQL(tableName, getBool(params, "if_exists"))

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("表 %s 删除成功", tableName),
	}, nil
}

func handleCreateIndex(params map[string]interface{}) (interface{}, error) {
	indexName := getString(params, "index_name")
	tableName := getString(params, "table_name")
	columns, ok := params["columns"].([]interface{})
	if !ok || len(columns) == 0 {
		return nil, fmt.Errorf("参数 columns 是必需的且不能为空")
	}

	sql, err := buildCreateIndexSQL(indexName, tableName, columns, getBool(params, "unique"))
	if err != nil {
		return nil, err
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("索引 %s 创建成功", indexName),
	}, nil
}

func handleDropIndex(params map[string]interface{}) (interface{}, error) {
	indexName := getString(params, "index_name")
	if indexName == "" {
		return nil, fmt.Errorf("参数 index_name 是必需的")
	}

	schema := getString(params, "schema")
	sql := buildDropIndexSQL(indexName, schema, getBool(params, "if_exists"))

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	disp := indexName
	if schema != "" && !strings.Contains(indexName, ".") {
		disp = schema + "." + indexName
	}
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("索引 %s 删除成功", disp),
	}, nil
}

func handleExecuteDDL(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "DDL语句执行成功",
	}, nil
}

func handleBatchCreateTables(params map[string]interface{}) (interface{}, error) {
	tables, ok := params["tables"].([]interface{})
	if !ok || len(tables) == 0 {
		return nil, fmt.Errorf("参数 tables 是必需的且不能为空")
	}

	atomic := getBoolOrDefault(params, "atomic", false)
	statements := make([]string, 0, len(tables))

	for i, t := range tables {
		tm, ok := t.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("tables[%d] 必须是对象", i)
		}
		tableName := getString(tm, "table_name")
		cols, ok := tm["columns"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("tables[%d].columns 是必需的", i)
		}
		sql, err := buildCreateTableSQL(tableName, cols)
		if err != nil {
			return nil, fmt.Errorf("tables[%d]: %w", i, err)
		}
		statements = append(statements, sql)
	}

	return runBatchDDL(statements, atomic)
}

func handleBatchCreateIndexes(params map[string]interface{}) (interface{}, error) {
	indexes, ok := params["indexes"].([]interface{})
	if !ok || len(indexes) == 0 {
		return nil, fmt.Errorf("参数 indexes 是必需的且不能为空")
	}

	atomic := getBoolOrDefault(params, "atomic", false)
	statements := make([]string, 0, len(indexes))

	for i, ix := range indexes {
		im, ok := ix.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("indexes[%d] 必须是对象", i)
		}
		indexName := getString(im, "index_name")
		tableName := getString(im, "table_name")
		cols, ok := im["columns"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("indexes[%d].columns 是必需的", i)
		}
		sql, err := buildCreateIndexSQL(indexName, tableName, cols, getBool(im, "unique"))
		if err != nil {
			return nil, fmt.Errorf("indexes[%d]: %w", i, err)
		}
		statements = append(statements, sql)
	}

	return runBatchDDL(statements, atomic)
}

func handleBatchDropTables(params map[string]interface{}) (interface{}, error) {
	names, err := stringSliceParam(params, "table_names")
	if err != nil {
		return nil, err
	}

	ifExists := getBool(params, "if_exists")
	atomic := getBoolOrDefault(params, "atomic", false)

	statements := make([]string, 0, len(names))
	for _, name := range names {
		statements = append(statements, buildDropTableSQL(name, ifExists))
	}

	return runBatchDDL(statements, atomic)
}

func handleBatchDropIndexes(params map[string]interface{}) (interface{}, error) {
	raw, ok := params["index_names"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("参数 index_names 是必需的且不能为空")
	}

	ifExists := getBool(params, "if_exists")
	atomic := getBoolOrDefault(params, "atomic", false)

	statements := make([]string, 0, len(raw))
	for i, v := range raw {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" {
			return nil, fmt.Errorf("index_names[%d] 不能为空", i)
		}
		statements = append(statements, buildDropIndexSQL(s, "", ifExists))
	}

	return runBatchDDL(statements, atomic)
}

func stringSliceParam(params map[string]interface{}, key string) ([]string, error) {
	raw, ok := params[key].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("参数 %s 是必需的且不能为空", key)
	}
	out := make([]string, len(raw))
	for i, v := range raw {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" {
			return nil, fmt.Errorf("%s[%d] 不能为空", key, i)
		}
		out[i] = s
	}
	return out, nil
}
