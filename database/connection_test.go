package database

import (
	"os"
	"testing"
)

// requireDB 检查是否存在可用凭据；无凭据时跳过集成测试。
// 本机开发环境可设置 DM_PASSWORD 等环境变量后运行。
func requireDB(t *testing.T) {
	t.Helper()
	if os.Getenv("DM_PASSWORD") == "" {
		t.Skip("未设置 DM_PASSWORD，跳过数据库集成测试")
	}
}

func TestConnectAndPing(t *testing.T) {
	requireDB(t)
	db, err := GetDB()
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping 失败: %v", err)
	}
}

func TestExecuteBatchInsert(t *testing.T) {
	requireDB(t)
	tbl := "T_MCP_TEST_BATCH"

	if err := ExecuteDDL("DROP TABLE IF EXISTS " + tbl); err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if err := ExecuteDDL("CREATE TABLE " + tbl + " (ID INT, NAME VARCHAR(100), SCORE DECIMAL(10,2))"); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	defer ExecuteDDL("DROP TABLE " + tbl)

	// 构造 10 行 × 3 列
	rows := make([][]interface{}, 10)
	for i := range rows {
		rows[i] = []interface{}{i + 1, "name_" + string(rune('a'+i)), float64(i) * 1.5}
	}

	affected, err := ExecuteBatchInsert(tbl, []string{"ID", "NAME", "SCORE"}, rows)
	if err != nil {
		t.Fatalf("批量插入失败: %v", err)
	}
	if affected != 10 {
		t.Errorf("影响行数应为 10, got %d", affected)
	}

	// 验证数据
	res, err := Query("SELECT COUNT(*) AS C FROM " + tbl)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(res) != 1 || res[0]["C"].(int64) != 10 {
		t.Errorf("插入行数校验失败: %v", res)
	}
}
