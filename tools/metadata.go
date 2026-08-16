package tools

import (
	"fmt"
	"strings"
)

func init() {
	registerMetadataTools()
}

func registerMetadataTools() {
	RegisterTool(ToolInfo{
		Name:        "list_databases",
		Category:    "metadata",
		Description: "列出达梦数据库服务器上的所有数据库",
		Params:      []string{},
	}, handleListDatabases)

	RegisterTool(ToolInfo{
		Name:        "list_schemas",
		Category:    "metadata",
		Description: "列出当前数据库的所有模式（Schema）",
		Params:      []string{},
	}, handleListSchemas)

	RegisterTool(ToolInfo{
		Name:        "list_tables",
		Category:    "metadata",
		Description: "列出表。参数: schema(可选)-模式名",
		Params:      []string{"schema"},
	}, handleListTables)

	RegisterTool(ToolInfo{
		Name:        "list_views",
		Category:    "metadata",
		Description: "列出视图。参数: schema(可选)-模式名",
		Params:      []string{"schema"},
	}, handleListViews)

	RegisterTool(ToolInfo{
		Name:        "describe_table",
		Category:    "metadata",
		Description: "获取表结构。参数: table_name-表名",
		Params:      []string{"table_name"},
	}, handleDescribeTable)

	RegisterTool(ToolInfo{
		Name:        "list_indexes",
		Category:    "metadata",
		Description: "列出表的索引。参数: table_name-表名",
		Params:      []string{"table_name"},
	}, handleListIndexes)

	RegisterTool(ToolInfo{
		Name:        "search_indexes",
		Category:    "metadata",
		Description: "索引目录检索。参数: owner_scope(可选,USER|ALL,默认USER), schema(可选,ALL时筛选索引属主), table_name(可选), index_name(可选), index_match(可选,exact|prefix|like,默认prefix)",
		Params:      []string{"owner_scope", "schema", "table_name", "index_name", "index_match"},
	}, handleSearchIndexes)

	RegisterTool(ToolInfo{
		Name:        "describe_index",
		Category:    "metadata",
		Description: "索引详情（元数据+索引列）。参数: index_name(必填), table_name(可选), owner_scope(可选,USER|ALL), schema(可选,ALL时必填索引属主)",
		Params:      []string{"index_name", "table_name", "owner_scope", "schema"},
	}, handleDescribeIndex)
}

func handleListDatabases(params map[string]interface{}) (interface{}, error) {
	sql := "SELECT NAME FROM V$DATABASE"
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"databases": results,
		"count":     len(results),
	}, nil
}

func handleListSchemas(params map[string]interface{}) (interface{}, error) {
	sql := "SELECT USERNAME AS SCHEMA_NAME FROM DBA_USERS ORDER BY USERNAME"
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"schemas": results,
		"count":   len(results),
	}, nil
}

func handleListTables(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	if schema == "" {
		schema = getEnvOrDefault("DM_SCHEMA", "")
	}

	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT TABLE_NAME, OWNER FROM ALL_TABLES WHERE OWNER = '%s' ORDER BY TABLE_NAME", schema)
	} else {
		sql = "SELECT TABLE_NAME FROM USER_TABLES ORDER BY TABLE_NAME"
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"tables": results,
		"count":  len(results),
	}, nil
}

func handleListViews(params map[string]interface{}) (interface{}, error) {
	schema := getString(params, "schema")
	if schema == "" {
		schema = getEnvOrDefault("DM_SCHEMA", "")
	}

	var sql string
	if schema != "" {
		sql = fmt.Sprintf("SELECT VIEW_NAME, OWNER FROM ALL_VIEWS WHERE OWNER = '%s' ORDER BY VIEW_NAME", schema)
	} else {
		sql = "SELECT VIEW_NAME FROM USER_VIEWS ORDER BY VIEW_NAME"
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"views": results,
		"count": len(results),
	}, nil
}

func handleDescribeTable(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	sql := fmt.Sprintf(`
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			DATA_LENGTH,
			DATA_PRECISION,
			DATA_SCALE,
			NULLABLE,
			DATA_DEFAULT
		FROM USER_TAB_COLUMNS
		WHERE TABLE_NAME = '%s'
		ORDER BY COLUMN_ID`, tableName)

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("表 %s 不存在或无权限访问", tableName)
	}

	return map[string]interface{}{
		"table_name": tableName,
		"columns":    results,
		"count":      len(results),
	}, nil
}

func handleListIndexes(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	sql := fmt.Sprintf(`
		SELECT
			INDEX_NAME,
			INDEX_TYPE,
			UNIQUENESS,
			STATUS
		FROM USER_INDEXES
		WHERE TABLE_NAME = '%s'
		ORDER BY INDEX_NAME`, tableName)

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"table_name": tableName,
		"indexes":    results,
		"count":      len(results),
	}, nil
}

func handleSearchIndexes(params map[string]interface{}) (interface{}, error) {
	ownerScope := strings.ToUpper(strings.TrimSpace(getString(params, "owner_scope")))
	if ownerScope == "" {
		ownerScope = "USER"
	}
	if ownerScope != "USER" && ownerScope != "ALL" {
		return nil, fmt.Errorf("owner_scope 必须是 USER 或 ALL")
	}

	schema := getString(params, "schema")
	tableName := getString(params, "table_name")
	indexName := getString(params, "index_name")
	match := getString(params, "index_match")
	if match == "" {
		match = "prefix"
	}

	var sqlStr string
	args := []interface{}{}

	if ownerScope == "USER" {
		sqlStr = `SELECT INDEX_NAME, TABLE_NAME, INDEX_TYPE, UNIQUENESS, STATUS FROM USER_INDEXES WHERE 1=1`
		if tableName != "" {
			sqlStr += fmt.Sprintf(" AND TABLE_NAME = :%d", len(args)+1)
			args = append(args, tableName)
		}
		var err error
		sqlStr, args, err = appendIndexNameSQL(sqlStr, args, "INDEX_NAME", indexName, match)
		if err != nil {
			return nil, err
		}
		sqlStr += " ORDER BY INDEX_NAME"
	} else {
		sqlStr = `SELECT OWNER AS INDEX_OWNER, INDEX_NAME, TABLE_NAME, TABLE_OWNER, INDEX_TYPE, UNIQUENESS, STATUS FROM ALL_INDEXES WHERE 1=1`
		if schema != "" {
			sqlStr += fmt.Sprintf(" AND OWNER = :%d", len(args)+1)
			args = append(args, schema)
		}
		if tableName != "" {
			sqlStr += fmt.Sprintf(" AND TABLE_NAME = :%d", len(args)+1)
			args = append(args, tableName)
		}
		var err error
		sqlStr, args, err = appendIndexNameSQL(sqlStr, args, "INDEX_NAME", indexName, match)
		if err != nil {
			return nil, err
		}
		sqlStr += " ORDER BY OWNER, INDEX_NAME"
	}

	results, err := queryDB(sqlStr, args...)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"owner_scope": ownerScope,
		"indexes":     results,
		"count":       len(results),
	}, nil
}

func appendIndexNameSQL(sql string, args []interface{}, col, indexName, match string) (string, []interface{}, error) {
	if indexName == "" {
		return sql, args, nil
	}
	pos := len(args) + 1
	switch match {
	case "exact":
		sql += fmt.Sprintf(" AND %s = :%d", col, pos)
		args = append(args, indexName)
	case "like":
		sql += fmt.Sprintf(" AND %s LIKE :%d", col, pos)
		args = append(args, indexName)
	case "prefix":
		sql += fmt.Sprintf(" AND %s LIKE :%d", col, pos)
		args = append(args, indexName+"%")
	default:
		return "", nil, fmt.Errorf("index_match 无效，请使用 exact、prefix 或 like")
	}
	return sql, args, nil
}

func handleDescribeIndex(params map[string]interface{}) (interface{}, error) {
	indexName := getString(params, "index_name")
	if indexName == "" {
		return nil, fmt.Errorf("参数 index_name 是必需的")
	}
	tableName := getString(params, "table_name")
	ownerScope := strings.ToUpper(strings.TrimSpace(getString(params, "owner_scope")))
	if ownerScope == "" {
		ownerScope = "USER"
	}
	schema := getString(params, "schema")

	if ownerScope == "ALL" && schema == "" {
		return nil, fmt.Errorf("owner_scope=ALL 时请提供 schema（索引属主 OWNER）")
	}

	var meta []map[string]interface{}
	var cols []map[string]interface{}
	var err error

	if ownerScope == "USER" {
		var metaSQL string
		var args []interface{}
		if tableName == "" {
			metaSQL = `SELECT INDEX_NAME, TABLE_NAME, INDEX_TYPE, UNIQUENESS, STATUS FROM USER_INDEXES WHERE INDEX_NAME = :1`
			args = []interface{}{indexName}
		} else {
			metaSQL = `SELECT INDEX_NAME, TABLE_NAME, INDEX_TYPE, UNIQUENESS, STATUS FROM USER_INDEXES WHERE INDEX_NAME = :1 AND TABLE_NAME = :2`
			args = []interface{}{indexName, tableName}
		}
		meta, err = queryDB(metaSQL, args...)
		if err != nil {
			return nil, err
		}
		if len(meta) == 0 {
			return nil, fmt.Errorf("未找到索引 %s", indexName)
		}
		if tableName == "" && len(meta) > 1 {
			return nil, fmt.Errorf("存在多个同名索引 %s，请指定 table_name", indexName)
		}

		var colSQL string
		var cargs []interface{}
		if tableName == "" {
			colSQL = `SELECT COLUMN_NAME, COLUMN_POSITION FROM USER_IND_COLUMNS WHERE INDEX_NAME = :1 ORDER BY COLUMN_POSITION`
			cargs = []interface{}{indexName}
		} else {
			colSQL = `SELECT COLUMN_NAME, COLUMN_POSITION FROM USER_IND_COLUMNS WHERE INDEX_NAME = :1 AND TABLE_NAME = :2 ORDER BY COLUMN_POSITION`
			cargs = []interface{}{indexName, tableName}
		}
		cols, err = queryDB(colSQL, cargs...)
		if err != nil {
			return nil, err
		}
	} else {
		var metaSQL string
		var args []interface{}
		if tableName == "" {
			metaSQL = `SELECT OWNER AS INDEX_OWNER, INDEX_NAME, TABLE_NAME, TABLE_OWNER, INDEX_TYPE, UNIQUENESS, STATUS FROM ALL_INDEXES WHERE OWNER = :1 AND INDEX_NAME = :2`
			args = []interface{}{schema, indexName}
		} else {
			metaSQL = `SELECT OWNER AS INDEX_OWNER, INDEX_NAME, TABLE_NAME, TABLE_OWNER, INDEX_TYPE, UNIQUENESS, STATUS FROM ALL_INDEXES WHERE OWNER = :1 AND INDEX_NAME = :2 AND TABLE_NAME = :3`
			args = []interface{}{schema, indexName, tableName}
		}
		meta, err = queryDB(metaSQL, args...)
		if err != nil {
			return nil, err
		}
		if len(meta) == 0 {
			return nil, fmt.Errorf("未找到索引 %s（属主 %s）", indexName, schema)
		}

		var colSQL string
		var cargs []interface{}
		if tableName == "" {
			colSQL = `SELECT COLUMN_NAME, COLUMN_POSITION FROM ALL_IND_COLUMNS WHERE INDEX_OWNER = :1 AND INDEX_NAME = :2 ORDER BY COLUMN_POSITION`
			cargs = []interface{}{schema, indexName}
		} else {
			colSQL = `SELECT COLUMN_NAME, COLUMN_POSITION FROM ALL_IND_COLUMNS WHERE INDEX_OWNER = :1 AND INDEX_NAME = :2 AND TABLE_NAME = :3 ORDER BY COLUMN_POSITION`
			cargs = []interface{}{schema, indexName, tableName}
		}
		cols, err = queryDB(colSQL, cargs...)
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"index_name":   indexName,
		"owner_scope":  ownerScope,
		"meta":         meta[0],
		"columns":      cols,
		"column_count": len(cols),
	}, nil
}
