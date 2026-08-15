package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// 基准测试需要连接本机达梦库；未设置 DM_PASSWORD 时跳过。
// 运行：DM_PASSWORD=xxx go test -bench=. -benchtime=1x ./database/

var benchTable = "T_MCP_BENCH"

func benchSetup(b *testing.B) {
	if os.Getenv("DM_PASSWORD") == "" {
		b.Skip("未设置 DM_PASSWORD，跳过基准测试")
	}
	if _, err := GetDB(); err != nil {
		b.Fatalf("连接失败: %v", err)
	}
	if err := ExecuteDDL("DROP TABLE IF EXISTS " + benchTable); err != nil {
		b.Fatalf("清理失败: %v", err)
	}
	if err := ExecuteDDL("CREATE TABLE " + benchTable + " (ID INT, NAME VARCHAR(100), AMOUNT DECIMAL(14,2), TS TIMESTAMP)"); err != nil {
		b.Fatalf("建表失败: %v", err)
	}
}

func benchTeardown() {
	ExecuteDDL("DROP TABLE IF EXISTS " + benchTable)
}

// makeRows 生成 n 行测试数据。
func makeRows(n int) [][]interface{} {
	rows := make([][]interface{}, n)
	for i := range rows {
		rows[i] = []interface{}{i, fmt.Sprintf("item-%d", i), float64(i) * 0.99}
	}
	return rows
}

// BenchmarkBatchInsertMultiRow 多行 VALUES 批量插入基准。
func BenchmarkBatchInsertMultiRow(b *testing.B) {
	benchSetup(b)
	defer benchTeardown()

	for _, n := range []int{1000, 10000, 100000} {
		rows := makeRows(n)
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ExecuteBatchInsert(benchTable, []string{"ID", "NAME", "AMOUNT"}, rows); err != nil {
					b.Fatalf("批量插入失败: %v", err)
				}
			}
		})
		// 每轮后清空数据
		if _, err := Execute("DELETE FROM " + benchTable); err != nil {
			b.Fatalf("清空失败: %v", err)
		}
	}
}

// BenchmarkBatchInsertLegacy 模拟 v1 旧式实现：逐条 INSERT（字符串拼接值）+ 单事务。
// 用于量化 v2 多行 VALUES 参数绑定方案的性能提升。
func BenchmarkBatchInsertLegacy(b *testing.B) {
	benchSetup(b)
	defer benchTeardown()

	legacyExec := func(tbl string, rows []map[string]interface{}) error {
		d, err := GetDB()
		if err != nil {
			return err
		}
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		for _, row := range rows {
			var cols, vals []string
			var args []interface{}
			i := 1
			for col, val := range row {
				cols = append(cols, col)
				vals = append(vals, fmt.Sprintf(":%d", i))
				args = append(args, val)
				i++
			}
			sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tbl, strings.Join(cols, ", "), strings.Join(vals, ", "))
			if _, err := tx.Exec(sql, args...); err != nil {
				tx.Rollback()
				return err
			}
		}
		return tx.Commit()
	}

	for _, n := range []int{1000, 10000} {
		rows := make([]map[string]interface{}, n)
		for i := range rows {
			rows[i] = map[string]interface{}{"ID": i, "NAME": fmt.Sprintf("item-%d", i), "AMOUNT": float64(i) * 0.99}
		}
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := legacyExec(benchTable, rows); err != nil {
					b.Fatalf("legacy 批量插入失败: %v", err)
				}
			}
		})
		if _, err := Execute("DELETE FROM " + benchTable); err != nil {
			b.Fatalf("清空失败: %v", err)
		}
	}
}

// BenchmarkQueryLarge 大表全表查询基准（先灌 100k 行）。
func BenchmarkQueryLarge(b *testing.B) {
	benchSetup(b)
	defer benchTeardown()

	rows := makeRows(100000)
	if _, err := ExecuteBatchInsert(benchTable, []string{"ID", "NAME", "AMOUNT"}, rows); err != nil {
		b.Fatalf("预灌数据失败: %v", err)
	}

	b.Run("full-scan-100k", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res, err := Query("SELECT * FROM " + benchTable)
			if err != nil {
				b.Fatalf("查询失败: %v", err)
			}
			if len(res) != 100000 {
				b.Fatalf("行数不符: %d", len(res))
			}
		}
	})

	b.Run("paged-100k", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res, err := Query("SELECT * FROM " + benchTable + " LIMIT 1000 OFFSET 0")
			if err != nil {
				b.Fatalf("分页查询失败: %v", err)
			}
			if len(res) != 1000 {
				b.Fatalf("分页行数不符: %d", len(res))
			}
		}
	})
}
