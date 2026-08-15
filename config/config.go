package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 达梦数据库连接与性能参数配置。
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Schema   string

	// 连接池性能参数（DM_MAX_*）
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数（保持热连接）
	ConnMaxLifetime time.Duration // 连接最大存活时长（避免陈旧连接）
	ConnMaxIdleTime time.Duration // 空闲连接回收时长

	// 执行性能参数（DM_*）
	QueryTimeout time.Duration // 单请求超时
	BatchSize    int           // 批量写入分块大小
	QueryLimit   int           // 查询默认行数上限

	// 驱动透传参数（DM_DRIVER_PARAMS）：追加到 DSN query 的额外驱动属性，
	// 如 rowPrefetch=100&stmtPoolMaxSize=50（参考达梦 Go 驱动属性）。
	DriverParams string

	// 外部工具路径（DM_*_PATH）
	FldrPath string // dmfldr 批量导入
	ExpPath  string // dexp 逻辑导出
	ImpPath  string // dimp 逻辑导入
	RmanPath string // dmrman 物理备份
}

// LoadConfig 从 DM_* 环境变量加载配置。
func LoadConfig() *Config {
	cfg := &Config{
		Host:            getEnvOrDefault("DM_HOST", "localhost"),
		Port:            getEnvAsInt("DM_PORT", 5236),
		User:            getEnvOrDefault("DM_USER", "SYSDBA"),
		Password:        getEnvOrDefault("DM_PASSWORD", ""),
		Database:        getEnvOrDefault("DM_DATABASE", "DAMENG"),
		Schema:          getEnvOrDefault("DM_SCHEMA", ""),
		MaxOpenConns:    getEnvAsInt("DM_MAX_OPEN_CONNS", 16),
		MaxIdleConns:    getEnvAsInt("DM_MAX_IDLE_CONNS", 16),
		ConnMaxLifetime: getEnvAsDuration("DM_CONN_MAX_LIFETIME", 30*time.Minute),
		ConnMaxIdleTime: getEnvAsDuration("DM_CONN_MAX_IDLE_TIME", 5*time.Minute),
		QueryTimeout:    getEnvAsDuration("DM_QUERY_TIMEOUT", 30*time.Second),
		BatchSize:       getEnvAsInt("DM_BATCH_SIZE", 500),
		QueryLimit:      getEnvAsInt("DM_QUERY_LIMIT", 1000),
		DriverParams:    getEnvOrDefault("DM_DRIVER_PARAMS", ""),
		FldrPath:        getEnvOrDefault("DM_FLDR_PATH", "dmfldr"),
		ExpPath:         getEnvOrDefault("DM_EXP_PATH", "dexp"),
		ImpPath:         getEnvOrDefault("DM_IMP_PATH", "dimp"),
		RmanPath:        getEnvOrDefault("DM_RMAN_PATH", "dmrman"),
	}

	// 参数合理性兜底
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 16
	}
	if cfg.MaxIdleConns <= 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.QueryLimit <= 0 {
		cfg.QueryLimit = 1000
	}

	// 日志输出到 stderr，避免污染 stdio 协议流（stdout 为 newline-delimited JSON）
	log.Printf("[配置] 主机: %s, 端口: %d, 用户: %s, 数据库: %s, 连接池: %d/%d\n",
		cfg.Host, cfg.Port, cfg.User, cfg.Database, cfg.MaxOpenConns, cfg.MaxIdleConns)

	return cfg
}

// GetDSN 生成达梦驱动 DSN。
//
// 注意：达梦一个实例即一个数据库，驱动 DSN 的 URL path 会被当作 schema 执行
// "set schema"，因此 DSN 不带 path；schema 通过 query 参数传递，
// 由驱动在每连接建立时自动执行，从而对连接池中所有连接生效。
func (c *Config) GetDSN() string {
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d", c.User, c.Password, c.Host, c.Port)
	var params []string
	if c.Schema != "" {
		params = append(params, "schema="+c.Schema)
	}
	if c.DriverParams != "" {
		params = append(params, c.DriverParams)
	}
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}
	return dsn
}

// IsValid 校验配置是否可用于连接。
func (c *Config) IsValid() bool {
	return c.Host != "" && c.Password != ""
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
		log.Printf("[警告] 环境变量 %s 的值 '%s' 不是有效整数，使用默认值 %d\n", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvAsDuration 解析时长（支持 "30s"、"5m"、"1h" 或纯数字秒）。
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(value); err == nil {
		return time.Duration(secs) * time.Second
	}
	log.Printf("[警告] 环境变量 %s 的值 '%s' 无法解析为时长，使用默认值 %v\n", key, value, defaultValue)
	return defaultValue
}