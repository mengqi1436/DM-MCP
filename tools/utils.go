package tools

import (
	"dm-mcp/database"
	"os"
	"strconv"
	"strings"
)

func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func getInt(params map[string]interface{}, key string, defaultValue int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	if v, ok := params[key].(int); ok {
		return v
	}
	if v, ok := params[key].(string); ok {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}
	return defaultValue
}

func getBool(params map[string]interface{}, key string) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return false
}

func getBoolOrDefault(params map[string]interface{}, key string, defaultValue bool) bool {
	if _, ok := params[key]; !ok {
		return defaultValue
	}
	return getBool(params, key)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func queryDB(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	return database.Query(sqlStr, args...)
}

func executeDB(sqlStr string, args ...interface{}) (int64, error) {
	return database.Execute(sqlStr, args...)
}

func executeDDLDB(sqlStr string) error {
	return database.ExecuteDDL(sqlStr)
}

func executeTransactionDB(statements []string) error {
	return database.ExecuteTransaction(statements)
}
