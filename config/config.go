package config

import (
	"os"
	"strconv"
)

// Config 达梦数据库连接配置
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	port := 5236
	if p := os.Getenv("DM_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	return &Config{
		Host:     getEnvOrDefault("DM_HOST", "localhost"),
		Port:     port,
		User:     getEnvOrDefault("DM_USER", "SYSDBA"),
		Password: getEnvOrDefault("DM_PASSWORD", ""),
		Database: getEnvOrDefault("DM_DATABASE", "DMDB"),
	}
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	return "dm://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.Database
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
