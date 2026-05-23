package tools


// getString 从参数中获取字符串值
func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

// getInt 从参数中获取整数值
func getInt(params map[string]interface{}, key string, defaultValue int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	if v, ok := params[key].(int); ok {
		return v
	}
	return defaultValue
}

// getBool 从参数中获取布尔值
func getBool(params map[string]interface{}, key string) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return false
}

// getBoolOrDefault 从参数中获取布尔值，未设置时返回 defaultValue
func getBoolOrDefault(params map[string]interface{}, key string, defaultValue bool) bool {
	if _, ok := params[key]; !ok {
		return defaultValue
	}
	return getBool(params, key)
}



