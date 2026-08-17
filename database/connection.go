package database

import (
	"context"
	"database/sql"
	"dm-mcp2/config"
	"fmt"
	"strings"
	"sync"

	_ "gitee.com/chunanyong/dm"
)

var (
	db      *sql.DB
	cfg     *config.Config
	once    sync.Once
	initErr error
)

func getCfg() *config.Config {
	if cfg == nil {
		cfg = config.LoadConfig()
	}
	return cfg
}

// GetDB 返回调优后的连接池（单例初始化）。
func GetDB() (*sql.DB, error) {
	once.Do(func() {
		c := config.LoadConfig()
		cfg = c
		if !c.IsValid() {
			initErr = fmt.Errorf("数据库配置无效：请设置 DM_HOST, DM_PASSWORD, DM_DATABASE 环境变量")
			return
		}
		d, err := sql.Open("dm", c.GetDSN())
		if err != nil {
			initErr = fmt.Errorf("打开数据库连接失败: %w", err)
			return
		}
		// 连接池性能调优
		d.SetMaxOpenConns(c.MaxOpenConns)
		d.SetMaxIdleConns(c.MaxIdleConns)
		d.SetConnMaxLifetime(c.ConnMaxLifetime)
		d.SetConnMaxIdleTime(c.ConnMaxIdleTime)

		if err := d.Ping(); err != nil {
			initErr = fmt.Errorf("数据库连接失败: %w", err)
			return
		}
		db = d
	})
	return db, initErr
}

// withTimeout 为单次操作创建带查询超时的 context。
func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), getCfg().QueryTimeout)
}

// Query 执行 SELECT，返回结果集（键为列名）。
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	d, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	ctx, cancel := withTimeout()
	defer cancel()

	rows, err := d.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// Execute 执行 DML，返回影响行数。
func Execute(sqlStr string, args ...interface{}) (int64, error) {
	d, err := GetDB()
	if err != nil {
		return 0, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	ctx, cancel := withTimeout()
	defer cancel()

	result, err := d.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("执行语句失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return affected, nil
}

// ExecuteDDL 执行 DDL（达梦 DDL 隐式提交）。
func ExecuteDDL(sqlStr string) error {
	d, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	ctx, cancel := withTimeout()
	defer cancel()

	if _, err := d.ExecContext(ctx, sqlStr); err != nil {
		return fmt.Errorf("执行DDL失败: %w", err)
	}
	return nil
}

// ExecuteTransaction 在单事务内顺序执行多条语句。
func ExecuteTransaction(statements []string) error {
	d, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	ctx, cancel := withTimeout()
	defer cancel()

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("执行语句失败 [%s]: %w", stmt, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// ExecuteBatchInsert 多行 VALUES 批量插入（参数绑定 + 预编译 + 单事务）。
//
// 将 rows 按列顺序与 columns 对齐，按 BatchSize 分块生成
// "INSERT INTO t (c1,c2) VALUES (:1,:2),(:3,:4),..." 语句；
// 语句文本对固定大小的块是相同的，可命中预编译语句缓存。
func ExecuteBatchInsert(table string, columns []string, rows [][]interface{}) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	d, err := GetDB()
	if err != nil {
		return 0, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	c := getCfg()

	colList := strings.Join(columns, ", ")
	perStmt := c.BatchSize
	if perStmt <= 0 {
		perStmt = 500
	}

	ctx, cancel := withTimeout()
	defer cancel()

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %w", err)
	}

	var total int64
	for start := 0; start < len(rows); start += perStmt {
		end := start + perStmt
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]

		var sb strings.Builder
		sb.WriteString("INSERT INTO ")
		sb.WriteString(table)
		sb.WriteString(" (")
		sb.WriteString(colList)
		sb.WriteString(") VALUES ")
		args := make([]interface{}, 0, len(chunk)*len(columns))
		idx := 1
		for ri, row := range chunk {
			if ri > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(")
			for ci := range columns {
				if ci > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(&sb, ":%d", idx)
				idx++
				args = append(args, row[ci])
			}
			sb.WriteString(")")
		}

		res, err := tx.ExecContext(ctx, sb.String(), args...)
		if err != nil {
			tx.Rollback()
			return total, fmt.Errorf("批量插入第 %d-%d 行失败: %w", start+1, end, err)
		}
		n, _ := res.RowsAffected()
		total += n
	}

	if err := tx.Commit(); err != nil {
		return total, fmt.Errorf("提交事务失败: %w", err)
	}
	return total, nil
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for rows.Next() {
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// Close 关闭数据库连接池。
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
