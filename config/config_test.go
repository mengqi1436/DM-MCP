package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// 清空相关环境变量，验证默认值
	for _, k := range []string{"DM_HOST", "DM_PORT", "DM_USER", "DM_PASSWORD", "DM_DATABASE", "DM_SCHEMA",
		"DM_MAX_OPEN_CONNS", "DM_MAX_IDLE_CONNS", "DM_CONN_MAX_LIFETIME", "DM_CONN_MAX_IDLE_TIME",
		"DM_QUERY_TIMEOUT", "DM_BATCH_SIZE", "DM_QUERY_LIMIT", "DM_DRIVER_PARAMS"} {
		os.Unsetenv(k)
	}

	c := LoadConfig()
	if c.Host != "localhost" {
		t.Errorf("Host 默认值应为 localhost, got %s", c.Host)
	}
	if c.Port != 5236 {
		t.Errorf("Port 默认值应为 5236, got %d", c.Port)
	}
	if c.User != "SYSDBA" {
		t.Errorf("User 默认值应为 SYSDBA, got %s", c.User)
	}
	if c.Database != "DAMENG" {
		t.Errorf("Database 默认值应为 DAMENG (修正 v1 的 DMDB), got %s", c.Database)
	}
	if c.MaxOpenConns != 16 || c.MaxIdleConns != 16 {
		t.Errorf("连接池默认值应为 16/16, got %d/%d", c.MaxOpenConns, c.MaxIdleConns)
	}
	if c.QueryLimit != 1000 || c.BatchSize != 500 {
		t.Errorf("执行参数默认值错误: limit=%d batch=%d", c.QueryLimit, c.BatchSize)
	}
	if c.QueryTimeout != 30*time.Second {
		t.Errorf("QueryTimeout 默认值应为 30s, got %v", c.QueryTimeout)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	os.Setenv("DM_HOST", "192.168.1.10")
	os.Setenv("DM_PORT", "5237")
	os.Setenv("DM_USER", "TESTUSER")
	os.Setenv("DM_PASSWORD", "secret")
	os.Setenv("DM_DATABASE", "TESTDB")
	os.Setenv("DM_SCHEMA", "APP")
	os.Setenv("DM_MAX_OPEN_CONNS", "32")
	os.Setenv("DM_CONN_MAX_LIFETIME", "45m")
	defer func() {
		for _, k := range []string{"DM_HOST", "DM_PORT", "DM_USER", "DM_PASSWORD", "DM_DATABASE", "DM_SCHEMA", "DM_MAX_OPEN_CONNS", "DM_CONN_MAX_LIFETIME"} {
			os.Unsetenv(k)
		}
	}()

	c := LoadConfig()
	if c.Host != "192.168.1.10" || c.Port != 5237 || c.User != "TESTUSER" || c.Database != "TESTDB" {
		t.Errorf("环境变量加载错误: %+v", c)
	}
	if c.Schema != "APP" {
		t.Errorf("Schema 应为 APP, got %s", c.Schema)
	}
	if c.MaxOpenConns != 32 {
		t.Errorf("MaxOpenConns 应为 32, got %d", c.MaxOpenConns)
	}
	if c.ConnMaxLifetime != 45*time.Minute {
		t.Errorf("ConnMaxLifetime 应为 45m, got %v", c.ConnMaxLifetime)
	}
	if !c.IsValid() {
		t.Error("带密码配置应有效")
	}
}

func TestGetDSN(t *testing.T) {
	c := &Config{
		Host: "localhost", Port: 5236, User: "SYSDBA",
		Password: "pw", Database: "DAMENG",
	}
	// 达梦 DSN 不带 path（path 会被驱动当作 schema）
	want := "dm://SYSDBA:pw@localhost:5236"
	if got := c.GetDSN(); got != want {
		t.Errorf("GetDSN 无参数时错误: got %s, want %s", got, want)
	}

	// 带 schema 时应追加 query 参数
	c.Schema = "APP"
	if got := c.GetDSN(); got != "dm://SYSDBA:pw@localhost:5236?schema=APP" {
		t.Errorf("GetDSN 带 schema 错误: got %s", got)
	}

	// 带驱动参数时应追加
	c.DriverParams = "rowPrefetch=100&stmtPoolMaxSize=50"
	if got := c.GetDSN(); got != "dm://SYSDBA:pw@localhost:5236?schema=APP&rowPrefetch=100&stmtPoolMaxSize=50" {
		t.Errorf("GetDSN 带驱动参数错误: got %s", got)
	}
}

func TestGetEnvAsDurationInvalid(t *testing.T) {
	os.Setenv("DM_QUERY_TIMEOUT", "abc")
	defer os.Unsetenv("DM_QUERY_TIMEOUT")
	if d := getEnvAsDuration("DM_QUERY_TIMEOUT", 30*time.Second); d != 30*time.Second {
		t.Errorf("非法时长应回退默认值, got %v", d)
	}
}
