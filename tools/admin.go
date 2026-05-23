package tools

import (
	"dm-mcp/database"
	"fmt"
)

func init() {
	registerAdminTools()
}

func registerAdminTools() {
	// list_users - 列出所有用户
	RegisterTool(ToolInfo{
		Name:        "list_users",
		Category:    "admin",
		Description: "列出达梦数据库的所有用户",
		Params:      []string{},
	}, handleListUsers)

	// table_statistics - 获取表统计信息
	RegisterTool(ToolInfo{
		Name:        "table_statistics",
		Category:    "admin",
		Description: "获取表统计信息(行数、大小等)。参数: table_name-表名",
		Params:      []string{"table_name"},
	}, handleTableStatistics)

	// database_info - 获取数据库信息
	RegisterTool(ToolInfo{
		Name:        "database_info",
		Category:    "admin",
		Description: "获取达梦数据库服务器的基本信息（版本、实例名等）",
		Params:      []string{},
	}, handleDatabaseInfo)
}

func handleListUsers(params map[string]interface{}) (interface{}, error) {
	sql := `
		SELECT 
			USERNAME,
			ACCOUNT_STATUS,
			CREATED,
			DEFAULT_TABLESPACE
		FROM DBA_USERS 
		ORDER BY USERNAME`

	results, err := database.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %v", err)
	}

	return map[string]interface{}{
		"users": results,
		"count": len(results),
	}, nil
}

func handleTableStatistics(params map[string]interface{}) (interface{}, error) {
	tableName := getString(params, "table_name")
	if tableName == "" {
		return nil, fmt.Errorf("参数 table_name 是必需的")
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) AS ROW_COUNT FROM %s", tableName)
	countResults, err := database.Query(countSQL)
	if err != nil {
		return nil, fmt.Errorf("获取表统计信息失败: %v", err)
	}

	statsSQL := fmt.Sprintf(`
		SELECT 
			TABLE_NAME,
			NUM_ROWS,
			BLOCKS,
			LAST_ANALYZED
		FROM USER_TABLES 
		WHERE TABLE_NAME = '%s'`, tableName)

	statsResults, _ := database.Query(statsSQL)

	result := map[string]interface{}{
		"table_name": tableName,
	}

	if len(countResults) > 0 {
		result["actual_row_count"] = countResults[0]["ROW_COUNT"]
	}

	if len(statsResults) > 0 {
		result["estimated_rows"] = statsResults[0]["NUM_ROWS"]
		result["blocks"] = statsResults[0]["BLOCKS"]
		result["last_analyzed"] = statsResults[0]["LAST_ANALYZED"]
	}

	return result, nil
}

func handleDatabaseInfo(params map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{}

	versionSQL := "SELECT * FROM V$VERSION"
	versionResults, err := database.Query(versionSQL)
	if err != nil {
		versionSQL = "SELECT BANNER FROM V$VERSION"
		versionResults, _ = database.Query(versionSQL)
	}
	if len(versionResults) > 0 {
		result["version"] = versionResults
	}

	instanceSQL := "SELECT * FROM V$INSTANCE"
	instanceResults, _ := database.Query(instanceSQL)
	if len(instanceResults) > 0 {
		result["instance"] = instanceResults
	}

	dbSQL := "SELECT * FROM V$DATABASE"
	dbResults, _ := database.Query(dbSQL)
	if len(dbResults) > 0 {
		result["database"] = dbResults
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("无法获取数据库信息，请检查权限")
	}

	return result, nil
}
