package tools

import (
	"fmt"
	"strings"
)

func init() {
	registerSchemaObjectTools()
}

func registerSchemaObjectTools() {
	RegisterTool(ToolInfo{
		Name:        "list_sequences",
		Category:    "metadata",
		Description: "列出序列。参数: schema(可选)",
		Params:      []string{"schema"},
	}, handleListSequences)

	RegisterTool(ToolInfo{
		Name:        "list_synonyms",
		Category:    "metadata",
		Description: "列出同义词。参数: schema(可选)",
		Params:      []string{"schema"},
	}, handleListSynonyms)

	RegisterTool(ToolInfo{
		Name:        "list_procedures",
		Category:    "metadata",
		Description: "列出存储过程。参数: schema(可选)",
		Params:      []string{"schema"},
	}, handleListProcedures)

	RegisterTool(ToolInfo{
		Name:        "list_functions",
		Category:    "metadata",
		Description: "列出函数。参数: schema(可选)",
		Params:      []string{"schema"},
	}, handleListFunctions)

	RegisterTool(ToolInfo{
		Name:        "list_packages",
		Category:    "metadata",
		Description: "列出包。参数: schema(可选)",
		Params:      []string{"schema"},
	}, handleListPackages)

	RegisterTool(ToolInfo{
		Name:        "list_triggers",
		Category:    "metadata",
		Description: "列出触发器。参数: schema(可选), table_name(可选)",
		Params:      []string{"schema", "table_name"},
	}, handleListTriggers)

	RegisterTool(ToolInfo{
		Name:        "list_constraints",
		Category:    "metadata",
		Description: "列出表约束（主键、外键、唯一、检查）。参数: table_name(必填), schema(可选)",
		Params:      []string{"table_name", "schema"},
	}, handleListConstraints)

	RegisterTool(ToolInfo{
		Name:        "list_table_partitions",
		Category:    "metadata",
		Description: "列出表分区信息。参数: table_name(必填), schema(可选)",
		Params:      []string{"table_name", "schema"},
	}, handleListTablePartitions)

	RegisterTool(ToolInfo{
		Name:        "get_table_ddl",
		Category:    "metadata",
		Description: "导出建表DDL语句（含索引和约束）。参数: table_name(必填), schema(可选)",
		Params:      []string{"table_name", "schema"},
	}, handleGetTableDDL)

	RegisterTool(ToolInfo{
		Name:        "batch_describe_tables",
		Category:    "metadata",
		Description: "批量获取多表结构。参数: table_names(必填)-表名数组, schema(可选)",
		Params:      []string{"table_names", "schema"},
	}, handleBatchDescribeTables)

	RegisterTool(ToolInfo{
		Name:        "create_view",
		Category:    "ddl",
		Description: "创建视图。参数: view_name(必填), sql(必填)-SELECT语句, or_replace(可选)",
		Params:      []string{"view_name", "sql", "or_replace"},
	}, handleCreateView)

	RegisterTool(ToolInfo{
		Name:        "drop_view",
		Category:    "ddl",
		Description: "删除视图。参数: view_name(必填), if_exists(可选)",
		Params:      []string{"view_name", "if_exists"},
	}, handleDropView)

	RegisterTool(ToolInfo{
		Name:        "create_sequence",
		Category:    "ddl",
		Description: "创建序列。参数: seq_name(必填), start_with(可选,默认1), increment_by(可选,默认1), max_value(可选), cache_size(可选)",
		Params:      []string{"seq_name", "start_with", "increment_by", "max_value", "cache_size"},
	}, handleCreateSequence)

	RegisterTool(ToolInfo{
		Name:        "drop_sequence",
		Category:    "ddl",
		Description: "删除序列。参数: seq_name(必填), if_exists(可选)",
		Params:      []string{"seq_name", "if_exists"},
	}, handleDropSequence)
}

func handleListSequences(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT SEQUENCE_NAME, MIN_VALUE, MAX_VALUE, INCREMENT_BY, LAST_NUMBER FROM ALL_SEQUENCES WHERE SEQUENCE_OWNER = '%s' ORDER BY SEQUENCE_NAME", schema)
	} else {
		sql = "SELECT SEQUENCE_NAME, MIN_VALUE, MAX_VALUE, INCREMENT_BY, LAST_NUMBER FROM USER_SEQUENCES ORDER BY SEQUENCE_NAME"
	}
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"sequences": results, "count": len(results)}, nil
}

func handleListSynonyms(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT SYNONYM_NAME, TABLE_OWNER, TABLE_NAME FROM ALL_SYNONYMS WHERE OWNER = '%s' ORDER BY SYNONYM_NAME", schema)
	} else {
		sql = "SELECT SYNONYM_NAME, TABLE_OWNER, TABLE_NAME FROM USER_SYNONYMS ORDER BY SYNONYM_NAME"
	}
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"synonyms": results, "count": len(results)}, nil
}

func handleListProcedures(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM ALL_OBJECTS WHERE OWNER = '%s' AND OBJECT_TYPE = 'PROCEDURE' ORDER BY OBJECT_NAME", schema)
	} else {
		sql = "SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM USER_OBJECTS WHERE OBJECT_TYPE = 'PROCEDURE' ORDER BY OBJECT_NAME"
	}
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"procedures": results, "count": len(results)}, nil
}

func handleListFunctions(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM ALL_OBJECTS WHERE OWNER = '%s' AND OBJECT_TYPE = 'FUNCTION' ORDER BY OBJECT_NAME", schema)
	} else {
		sql = "SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM USER_OBJECTS WHERE OBJECT_TYPE = 'FUNCTION' ORDER BY OBJECT_NAME"
	}
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"functions": results, "count": len(results)}, nil
}

func handleListPackages(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM ALL_OBJECTS WHERE OWNER = '%s' AND OBJECT_TYPE = 'PACKAGE' ORDER BY OBJECT_NAME", schema)
	} else {
		sql = "SELECT OBJECT_NAME, STATUS, CREATED, LAST_DDL_TIME FROM USER_OBJECTS WHERE OBJECT_TYPE = 'PACKAGE' ORDER BY OBJECT_NAME"
	}
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"packages": results, "count": len(results)}, nil
}

func handleListTriggers(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	tableName := getString(params, "table_name")

	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT TRIGGER_NAME, TRIGGER_TYPE, TRIGGERING_EVENT, TABLE_NAME, STATUS FROM ALL_TRIGGERS WHERE OWNER = '%s'", schema)
	} else {
		sql = "SELECT TRIGGER_NAME, TRIGGER_TYPE, TRIGGERING_EVENT, TABLE_NAME, STATUS FROM USER_TRIGGERS WHERE 1=1"
	}
	if tableName != "" {
		sql += fmt.Sprintf(" AND TABLE_NAME = '%s'", tableName)
	}
	sql += " ORDER BY TRIGGER_NAME"

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"triggers": results, "count": len(results)}, nil
}

func handleListConstraints(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}
	schema := getString(params, "schema")

	var sql string
	if schema != "" {
		sql = fmt.Sprintf(`SELECT c.CONSTRAINT_NAME, c.CONSTRAINT_TYPE, c.STATUS, c.R_CONSTRAINT_NAME,
			cc.COLUMN_NAME, cc.POSITION
			FROM ALL_CONSTRAINTS c
			JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
			WHERE c.OWNER = '%s' AND c.TABLE_NAME = '%s'
			ORDER BY c.CONSTRAINT_TYPE, c.CONSTRAINT_NAME, cc.POSITION`, schema, tableName)
	} else {
		sql = fmt.Sprintf(`SELECT c.CONSTRAINT_NAME, c.CONSTRAINT_TYPE, c.STATUS, c.R_CONSTRAINT_NAME,
			cc.COLUMN_NAME, cc.POSITION
			FROM USER_CONSTRAINTS c
			JOIN USER_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
			WHERE c.TABLE_NAME = '%s'
			ORDER BY c.CONSTRAINT_TYPE, c.CONSTRAINT_NAME, cc.POSITION`, tableName)
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"table_name": tableName, "constraints": results, "count": len(results)}, nil
}

func handleListTablePartitions(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}
	schema := getString(params, "schema")

	var sql string
	if schema != "" {
		sql = fmt.Sprintf(`SELECT PARTITION_NAME, HIGH_VALUE, NUM_ROWS, LAST_ANALYZED
			FROM ALL_TAB_PARTITIONS WHERE TABLE_OWNER = '%s' AND TABLE_NAME = '%s'
			ORDER BY PARTITION_POSITION`, schema, tableName)
	} else {
		sql = fmt.Sprintf(`SELECT PARTITION_NAME, HIGH_VALUE, NUM_ROWS, LAST_ANALYZED
			FROM USER_TAB_PARTITIONS WHERE TABLE_NAME = '%s'
			ORDER BY PARTITION_POSITION`, tableName)
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"table_name": tableName, "partitions": results, "count": len(results)}, nil
}

func handleGetTableDDL(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}
	schema := getString(params, "schema")

	qualifiedTable := tableName
	colView := "USER_TAB_COLUMNS"
	ownerPredicate := ""
	if schema != "" {
		qualifiedTable = schema + "." + tableName
		colView = "ALL_TAB_COLUMNS"
		ownerPredicate = fmt.Sprintf("AND OWNER = '%s'", schema)
	}

	colSQL := fmt.Sprintf(`SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION, DATA_SCALE, NULLABLE, DATA_DEFAULT
		FROM %s WHERE TABLE_NAME = '%s' %s ORDER BY COLUMN_ID`, colView, tableName, ownerPredicate)
	columns, err := queryDB(colSQL)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("表 %s 不存在或无权限访问", qualifiedTable)
	}

	var colDefs []string
	for _, col := range columns {
		colName, _ := col["COLUMN_NAME"].(string)
		dataType, _ := col["DATA_TYPE"].(string)
		def := colName + " " + dataType

		if dataLength, ok := col["DATA_LENGTH"].(int64); ok && dataLength > 0 {
			switch dataType {
			case "VARCHAR", "VARCHAR2", "CHAR", "NVARCHAR2", "NCHAR":
				def += fmt.Sprintf("(%d)", dataLength)
			}
		}
		if precision, ok := col["DATA_PRECISION"].(int64); ok && precision > 0 {
			if scale, ok := col["DATA_SCALE"].(int64); ok && scale > 0 {
				def += fmt.Sprintf("(%d,%d)", precision, scale)
			} else {
				def += fmt.Sprintf("(%d)", precision)
			}
		}
		if nullable, _ := col["NULLABLE"].(string); nullable == "N" {
			def += " NOT NULL"
		}
		if defaultVal, _ := col["DATA_DEFAULT"].(string); defaultVal != "" {
			def += " DEFAULT " + strings.TrimSpace(defaultVal)
		}
		colDefs = append(colDefs, "  "+def)
	}

	pkCols := getPKColumns(tableName, schema)
	if len(pkCols) > 0 {
		colDefs = append(colDefs, "  PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (\n%s\n)", qualifiedTable, strings.Join(colDefs, ",\n"))

	indexSQLs := getIndexDDLs(tableName, schema)

	return map[string]interface{}{
		"table_name":  qualifiedTable,
		"create_sql":  createSQL,
		"index_sqls":  indexSQLs,
		"column_count": len(columns),
	}, nil
}

func getPKColumns(tableName, schema string) []string {
	consCols := "USER_CONS_COLUMNS"
	constraints := "USER_CONSTRAINTS"
	ownerJoin := ""
	ownerPred := ""
	if schema != "" {
		consCols = "ALL_CONS_COLUMNS"
		constraints = "ALL_CONSTRAINTS"
		ownerJoin = "AND c.OWNER = cc.OWNER"
		ownerPred = fmt.Sprintf("AND c.OWNER = '%s'", schema)
	}
	sql := fmt.Sprintf(`SELECT cc.COLUMN_NAME FROM %s cc JOIN %s c ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME %s
		WHERE c.TABLE_NAME = '%s' AND c.CONSTRAINT_TYPE = 'P' %s ORDER BY cc.POSITION`,
		consCols, constraints, ownerJoin, tableName, ownerPred)
	results, err := queryDB(sql)
	if err != nil {
		return nil
	}
	var cols []string
	for _, r := range results {
		if name, ok := r["COLUMN_NAME"].(string); ok {
			cols = append(cols, name)
		}
	}
	return cols
}

func getIndexDDLs(tableName, schema string) []string {
	idxView := "USER_INDEXES"
	colView := "USER_IND_COLUMNS"
	ownerPred := ""
	colOwnerPred := ""
	if schema != "" {
		idxView = "ALL_INDEXES"
		colView = "ALL_IND_COLUMNS"
		ownerPred = fmt.Sprintf("AND OWNER = '%s'", schema)
		colOwnerPred = fmt.Sprintf("AND INDEX_OWNER = '%s'", schema)
	}
	sql := fmt.Sprintf(`SELECT INDEX_NAME, UNIQUENESS FROM %s WHERE TABLE_NAME = '%s' AND INDEX_NAME NOT LIKE 'SYS_%%' %s`,
		idxView, tableName, ownerPred)
	results, err := queryDB(sql)
	if err != nil {
		return nil
	}

	qualifiedTable := tableName
	if schema != "" {
		qualifiedTable = schema + "." + tableName
	}

	var ddls []string
	for _, r := range results {
		idxName, _ := r["INDEX_NAME"].(string)
		uniqueness, _ := r["UNIQUENESS"].(string)

		colSQL := fmt.Sprintf(`SELECT COLUMN_NAME FROM %s WHERE INDEX_NAME = '%s' %s ORDER BY COLUMN_POSITION`,
			colView, idxName, colOwnerPred)
		colResults, err := queryDB(colSQL)
		if err != nil || len(colResults) == 0 {
			continue
		}
		var cols []string
		for _, cr := range colResults {
			if cn, ok := cr["COLUMN_NAME"].(string); ok {
				cols = append(cols, cn)
			}
		}
		unique := ""
		if uniqueness == "UNIQUE" {
			unique = "UNIQUE "
		}
		ddls = append(ddls, fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, idxName, qualifiedTable, strings.Join(cols, ", ")))
	}
	return ddls
}

func handleBatchDescribeTables(params map[string]interface{}) (interface{}, error) {
	names, err := stringSliceParam(params, "table_names")
	if err != nil {
		return nil, err
	}
	schema := getString(params, "schema")

	results := make([]map[string]interface{}, 0, len(names))
	for _, tableName := range names {
		colView := "USER_TAB_COLUMNS"
		ownerPred := ""
		if schema != "" {
			colView = "ALL_TAB_COLUMNS"
			ownerPred = fmt.Sprintf("AND OWNER = '%s'", schema)
		}
		sql := fmt.Sprintf(`SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION, DATA_SCALE, NULLABLE
			FROM %s WHERE TABLE_NAME = '%s' %s ORDER BY COLUMN_ID`, colView, tableName, ownerPred)
		cols, err := queryDB(sql)
		item := map[string]interface{}{"table_name": tableName}
		if err != nil {
			item["error"] = err.Error()
		} else if len(cols) == 0 {
			item["error"] = "表不存在或无权限"
		} else {
			item["columns"] = cols
			item["column_count"] = len(cols)
		}
		results = append(results, item)
	}

	return map[string]interface{}{"total": len(names), "results": results}, nil
}

func handleCreateView(params map[string]interface{}) (interface{}, error) {
	viewName := getString(params, "view_name")
	if viewName == "" {
		return nil, fmt.Errorf("参数 view_name 是必需的")
	}
	selectSQL := getString(params, "sql")
	if selectSQL == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	ddl := "CREATE "
	if getBool(params, "or_replace") {
		ddl += "OR REPLACE "
	}
	ddl += "VIEW " + viewName + " AS " + selectSQL

	if err := executeDDLDB(ddl); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("视图 %s 创建成功", viewName)}, nil
}

func handleDropView(params map[string]interface{}) (interface{}, error) {
	viewName := getString(params, "view_name")
	if viewName == "" {
		return nil, fmt.Errorf("参数 view_name 是必需的")
	}
	sql := "DROP VIEW "
	if getBool(params, "if_exists") {
		sql += "IF EXISTS "
	}
	sql += viewName

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("视图 %s 删除成功", viewName)}, nil
}

func handleCreateSequence(params map[string]interface{}) (interface{}, error) {
	seqName := getString(params, "seq_name")
	if seqName == "" {
		return nil, fmt.Errorf("参数 seq_name 是必需的")
	}

	sql := "CREATE SEQUENCE " + seqName
	if v := getInt(params, "start_with", 0); v != 0 {
		sql += fmt.Sprintf(" START WITH %d", v)
	}
	if v := getInt(params, "increment_by", 0); v != 0 {
		sql += fmt.Sprintf(" INCREMENT BY %d", v)
	}
	if v := getInt(params, "max_value", 0); v != 0 {
		sql += fmt.Sprintf(" MAXVALUE %d", v)
	}
	if v := getInt(params, "cache_size", 0); v != 0 {
		sql += fmt.Sprintf(" CACHE %d", v)
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("序列 %s 创建成功", seqName)}, nil
}

func handleDropSequence(params map[string]interface{}) (interface{}, error) {
	seqName := getString(params, "seq_name")
	if seqName == "" {
		return nil, fmt.Errorf("参数 seq_name 是必需的")
	}
	sql := "DROP SEQUENCE "
	if getBool(params, "if_exists") {
		sql += "IF EXISTS "
	}
	sql += seqName

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("序列 %s 删除成功", seqName)}, nil
}
