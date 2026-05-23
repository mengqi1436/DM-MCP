package tools

import (
	"dm-mcp/database"
	"fmt"
)

func init() {
	registerAdvancedTools()
}

func registerAdvancedTools() {
	// execute_transaction - 事务执行多条语句
	RegisterTool(ToolInfo{
		Name:        "execute_transaction",
		Category:    "advanced",
		Description: "事务执行多条SQL语句(全成功提交/任意失败回滚)。参数: statements-SQL语句数组。注意：若包含 DDL，达梦可能隐式提交，语义未必为严格单事务原子；批量 DDL 也可用 ddl/batch_execute_ddl 或 batch_* 工具（推荐 atomic=false）。",
		Params:      []string{"statements"},
	}, handleTransaction)

	// call_procedure - 调用存储过程
	RegisterTool(ToolInfo{
		Name:        "call_procedure",
		Category:    "advanced",
		Description: "调用存储过程。参数: procedure_name, params(可选)-参数数组",
		Params:      []string{"procedure_name", "params"},
	}, handleCallProcedure)

	// call_function - 调用函数
	RegisterTool(ToolInfo{
		Name:        "call_function",
		Category:    "advanced",
		Description: "调用函数并返回结果。参数: function_name, params(可选)-参数数组",
		Params:      []string{"function_name", "params"},
	}, handleCallFunction)

	// explain_plan - 分析执行计划
	RegisterTool(ToolInfo{
		Name:        "explain_plan",
		Category:    "advanced",
		Description: "分析SQL执行计划。参数: sql-SQL语句",
		Params:      []string{"sql"},
	}, handleExplainPlan)
}

func handleTransaction(params map[string]interface{}) (interface{}, error) {
	statements, ok := params["statements"].([]interface{})
	if !ok || len(statements) == 0 {
		return nil, fmt.Errorf("参数 statements 是必需的且不能为空")
	}

	stmts := make([]string, len(statements))
	for i, stmt := range statements {
		stmts[i] = fmt.Sprintf("%v", stmt)
	}

	err := database.ExecuteTransaction(stmts)
	if err != nil {
		return nil, fmt.Errorf("事务执行失败（已回滚）: %v", err)
	}

	return map[string]interface{}{
		"success":    true,
		"statements": len(stmts),
		"message":    fmt.Sprintf("事务执行成功，共执行 %d 条语句", len(stmts)),
	}, nil
}

func handleCallProcedure(params map[string]interface{}) (interface{}, error) {
	procedureName := getString(params, "procedure_name")
	if procedureName == "" {
		return nil, fmt.Errorf("参数 procedure_name 是必需的")
	}

	sql := fmt.Sprintf("CALL %s", procedureName)
	if paramsArr, ok := params["params"].([]interface{}); ok && len(paramsArr) > 0 {
		sql += "("
		for i, p := range paramsArr {
			if i > 0 {
				sql += ", "
			}
			sql += fmt.Sprintf("'%v'", p)
		}
		sql += ")"
	} else {
		sql += "()"
	}

	err := database.ExecuteDDL(sql)
	if err != nil {
		return nil, fmt.Errorf("调用存储过程失败: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("存储过程 %s 调用成功", procedureName),
	}, nil
}

func handleCallFunction(params map[string]interface{}) (interface{}, error) {
	functionName := getString(params, "function_name")
	if functionName == "" {
		return nil, fmt.Errorf("参数 function_name 是必需的")
	}

	sql := fmt.Sprintf("SELECT %s", functionName)
	if paramsArr, ok := params["params"].([]interface{}); ok && len(paramsArr) > 0 {
		sql += "("
		for i, p := range paramsArr {
			if i > 0 {
				sql += ", "
			}
			sql += fmt.Sprintf("'%v'", p)
		}
		sql += ")"
	} else {
		sql += "()"
	}
	sql += " AS RESULT FROM DUAL"

	results, err := database.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("调用函数失败: %v", err)
	}

	if len(results) > 0 {
		return map[string]interface{}{
			"success": true,
			"result":  results[0]["RESULT"],
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"result":  nil,
		"message": "函数执行成功但无返回值",
	}, nil
}

func handleExplainPlan(params map[string]interface{}) (interface{}, error) {
	sql := getString(params, "sql")
	if sql == "" {
		return nil, fmt.Errorf("参数 sql 是必需的")
	}

	explainSQL := fmt.Sprintf("EXPLAIN %s", sql)

	results, err := database.Query(explainSQL)
	if err != nil {
		return nil, fmt.Errorf("分析执行计划失败: %v", err)
	}

	return map[string]interface{}{
		"plan": results,
	}, nil
}
