package database

import (
	"database/sql"
	"dm-mcp/config"
	"fmt"
	"sync"

	_ "gitee.com/chunanyong/dm"
)

var (
	db   *sql.DB
	once sync.Once
)

// GetDB 获取数据库连接（单例模式）
func GetDB() (*sql.DB, error) {
	var err error
	once.Do(func() {
		cfg := config.LoadConfig()
		dsn := cfg.GetDSN()
		db, err = sql.Open("dm", dsn)
		if err != nil {
			return
		}
		// 配置连接池
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
	})
	return db, err
}

// Query 执行查询语句，返回结果集
func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	db, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// Execute 执行DML语句（INSERT/UPDATE/DELETE）
func Execute(sqlStr string, args ...interface{}) (int64, error) {
	db, err := GetDB()
	if err != nil {
		return 0, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("执行语句失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}

	return affected, nil
}

// ExecuteDDL 执行DDL语句
func ExecuteDDL(sqlStr string) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	_, err = db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("执行DDL失败: %w", err)
	}

	return nil
}

// ExecuteTransaction 在事务中执行多条语句
func ExecuteTransaction(statements []string) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("执行语句失败 [%s]: %w", stmt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// scanRows 将查询结果转换为 map 切片
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

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
