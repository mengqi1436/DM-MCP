package tools

import (
	"fmt"
)

func init() {
	registerMonitoringTools()
}

func registerMonitoringTools() {
	RegisterTool(ToolInfo{
		Name:        "active_sessions",
		Category:    "monitoring",
		Description: "查看当前活跃会话。参数: limit(可选,默认50)",
		Params:      []string{"limit"},
	}, handleActiveSessions)

	RegisterTool(ToolInfo{
		Name:        "lock_info",
		Category:    "monitoring",
		Description: "查看当前锁信息（阻塞关系）",
		Params:      []string{},
	}, handleLockInfo)

	RegisterTool(ToolInfo{
		Name:        "slow_queries",
		Category:    "monitoring",
		Description: "查看慢查询统计（V$SQL_STAT）。参数: limit(可选,默认20)",
		Params:      []string{"limit"},
	}, handleSlowQueries)

	RegisterTool(ToolInfo{
		Name:        "tablespace_usage",
		Category:    "monitoring",
		Description: "查看表空间使用率",
		Params:      []string{},
	}, handleTablespaceUsage)

	RegisterTool(ToolInfo{
		Name:        "instance_parameters",
		Category:    "monitoring",
		Description: "查看实例参数（dm.ini）。参数: name(可选)-参数名模糊搜索",
		Params:      []string{"name"},
	}, handleInstanceParameters)

	RegisterTool(ToolInfo{
		Name:        "session_memory",
		Category:    "monitoring",
		Description: "查看会话内存使用情况",
		Params:      []string{},
	}, handleSessionMemory)
}

func handleActiveSessions(params map[string]interface{}) (interface{}, error) {
	limit := getInt(params, "limit", 50)
	sql := fmt.Sprintf(`SELECT
		SESS_ID,
		USER_NAME,
		CLNT_IP,
		STATE,
		CREATE_TIME,
		LAST_SEND_TIME,
		SQL_TEXT
		FROM V$SESSIONS
		WHERE STATE = 'ACTIVE'
		ORDER BY CREATE_TIME
		LIMIT %d`, limit)

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"sessions": results, "count": len(results)}, nil
}

func handleLockInfo(params map[string]interface{}) (interface{}, error) {
	sql := `SELECT
		l.SESS_ID AS HOLDER_SESS_ID,
		s.USER_NAME AS HOLDER_USER,
		l.TABLE_ID,
		l.LOCK_TYPE,
		l.LOCK_MODE,
		w.SESS_ID AS WAITER_SESS_ID,
		ws.USER_NAME AS WAITER_USER
		FROM V$LOCK l
		LEFT JOIN V$SESSIONS s ON l.SESS_ID = s.SESS_ID
		LEFT JOIN V$LOCK w ON l.TABLE_ID = w.TABLE_ID AND w.SESS_ID != l.SESS_ID
		LEFT JOIN V$SESSIONS ws ON w.SESS_ID = ws.SESS_ID
		WHERE l.BLOCKED = 1
		ORDER BY l.SESS_ID`

	results, err := queryDB(sql)
	if err != nil {
		sql = `SELECT SESS_ID, TABLE_ID, LOCK_TYPE, LOCK_MODE, BLOCKED FROM V$LOCK WHERE BLOCKED = 1`
		results, err = queryDB(sql)
		if err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{"locks": results, "count": len(results)}, nil
}

func handleSlowQueries(params map[string]interface{}) (interface{}, error) {
	limit := getInt(params, "limit", 20)
	sql := fmt.Sprintf(`SELECT
		SQL_TEXT,
		EXECUTIONS,
		ELAPSED_TIME,
		CPU_TIME,
		LOGIC_READS,
		PHYSIC_READS
		FROM V$SQL_STAT
		WHERE ELAPSED_TIME > 0
		ORDER BY ELAPSED_TIME DESC
		LIMIT %d`, limit)

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"slow_queries": results, "count": len(results)}, nil
}

func handleTablespaceUsage(params map[string]interface{}) (interface{}, error) {
	sql := `SELECT
		t.TABLESPACE_NAME,
		d.TOTAL_MB,
		d.TOTAL_MB - NVL(f.FREE_MB, 0) AS USED_MB,
		NVL(f.FREE_MB, 0) AS FREE_MB,
		ROUND((d.TOTAL_MB - NVL(f.FREE_MB, 0)) * 100.0 / d.TOTAL_MB, 2) AS USED_PCT
		FROM DBA_TABLESPACES t
		JOIN (
			SELECT TABLESPACE_NAME, ROUND(SUM(BYTES)/1024/1024, 2) AS TOTAL_MB
			FROM DBA_DATA_FILES GROUP BY TABLESPACE_NAME
		) d ON t.TABLESPACE_NAME = d.TABLESPACE_NAME
		LEFT JOIN (
			SELECT TABLESPACE_NAME, ROUND(SUM(BYTES)/1024/1024, 2) AS FREE_MB
			FROM DBA_FREE_SPACE GROUP BY TABLESPACE_NAME
		) f ON t.TABLESPACE_NAME = f.TABLESPACE_NAME
		ORDER BY USED_PCT DESC`

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"tablespaces": results, "count": len(results)}, nil
}

func handleInstanceParameters(params map[string]interface{}) (interface{}, error) {
	name := getString(params, "name")

	var sql string
	if name != "" {
		sql = fmt.Sprintf("SELECT PARA_NAME, PARA_VALUE, DEFAULT_VALUE, PARA_TYPE FROM V$DM_INI WHERE PARA_NAME LIKE '%%%s%%' ORDER BY PARA_NAME", name)
	} else {
		sql = "SELECT PARA_NAME, PARA_VALUE, DEFAULT_VALUE, PARA_TYPE FROM V$DM_INI ORDER BY PARA_NAME LIMIT 200"
	}

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"parameters": results, "count": len(results)}, nil
}

func handleSessionMemory(params map[string]interface{}) (interface{}, error) {
	sql := `SELECT
		SESS_ID,
		USER_NAME,
		CLNT_IP,
		MEM_USED
		FROM V$SESSIONS
		WHERE MEM_USED > 0
		ORDER BY MEM_USED DESC
		LIMIT 50`

	results, err := queryDB(sql)
	if err != nil {
		sql = `SELECT SESS_ID, USER_NAME, CLNT_IP FROM V$SESSIONS LIMIT 50`
		results, err = queryDB(sql)
		if err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{"sessions": results, "count": len(results)}, nil
}
