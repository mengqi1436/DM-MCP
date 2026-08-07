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

func GetDB() (*sql.DB, error) {
	var err error
	once.Do(func() {
		cfg := config.LoadConfig()
		if !cfg.IsValid() {
			err = fmt.Errorf("数据库配置无效：请设置 DM_HOST, DM_PASSWORD, DM_DATABASE 环境变量")
			return
		}
		dsn := cfg.GetDSN()
		db, err = sql.Open("dm", dsn)
		if err != nil {
			return
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		if cfg.Schema != "" {
			_, err = db.Exec("ALTER SESSION SET CURRENT_SCHEMA = " + cfg.Schema)
			if err != nil {
				fmt.Printf("警告: 设置默认 Schema 失败: %v\n", err)
				err = nil
			}
		}
	})
	return db, err
}

func Query(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	d, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	rows, err := d.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

func Execute(sqlStr string, args ...interface{}) (int64, error) {
	d, err := GetDB()
	if err != nil {
		return 0, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	result, err := d.Exec(sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("执行语句失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}

	return affected, nil
}

func ExecuteDDL(sqlStr string) error {
	d, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	_, err = d.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("执行DDL失败: %w", err)
	}

	return nil
}

func ExecuteTransaction(statements []string) error {
	d, err := GetDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	tx, err := d.Begin()
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

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
