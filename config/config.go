package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Schema   string
}

func LoadConfig() *Config {
	port := getEnvAsInt("DM_PORT", 5236)

	cfg := &Config{
		Host:     getEnvOrDefault("DM_HOST", "localhost"),
		Port:     port,
		User:     getEnvOrDefault("DM_USER", "SYSDBA"),
		Password: getEnvOrDefault("DM_PASSWORD", ""),
		Database: getEnvOrDefault("DM_DATABASE", "DMDB"),
		Schema:   getEnvOrDefault("DM_SCHEMA", ""),
	}

	fmt.Printf("[配置] 主机: %s, 端口: %d, 用户: %s, 数据库: %s\n",
		cfg.Host, cfg.Port, cfg.User, cfg.Database)

	return cfg
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("dm://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.Database)
}

func (c *Config) IsValid() bool {
	return c.Host != "" && c.Database != "" && c.Password != ""
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
		fmt.Printf("[警告] 环境变量 %s 的值 '%s' 不是有效整数，使用默认值 %d\n", key, value, defaultValue)
	}
	return defaultValue
}
