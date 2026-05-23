package tools

import (
	"testing"
)

func TestBuildCSVControlFileWithColumnTypes(t *testing.T) {
	file := csvImportFile{
		CSVFile:     "E:/RHPT/TOPI1/BAS_COMPANY.csv",
		Schema:      "SYSDBA",
		Table:       "BAS_COMPANY",
		Columns:     []string{"ID", "COMPANYNAME", "STATUS", "CREATEDTIME", "ISDELETED"},
		ColumnTypes: []string{"CHAR", "CHAR", "INTEGER", "TIMESTAMP 'YYYY-MM-DD HH:MI:SS.FF'", "INTEGER"},
		NullIf:      "",
		Skip:        1,
	}
	options := csvImportOptions{
		Delimiter:     ",",
		EnclosedBy:    "\"",
		CharacterCode: "UTF-8",
		Rows:          50000,
		Direct:        true,
		IndexOption:   2,
		Mode:          "APPEND",
		Errors:        100,
	}

	control := buildCSVControlFile(options, file, "E:/RHPT/TOPI1/BAS_COMPANY.bad")

	// 验证控制文件包含字段类型
	assertContains(t, control, "ID CHAR")
	assertContains(t, control, "COMPANYNAME CHAR")
	assertContains(t, control, "STATUS INTEGER")
	assertContains(t, control, "CREATEDTIME TIMESTAMP 'YYYY-MM-DD HH:MI:SS.FF'")
	assertContains(t, control, "ISDELETED INTEGER")

	// 验证包含ERRORS参数
	assertContains(t, control, "ERRORS = 100")

	// 验证基本结构
	assertContains(t, control, "OPTIONS")
	assertContains(t, control, "SKIP = 1")
	assertContains(t, control, "ROWS = 50000")
	assertContains(t, control, "DIRECT = TRUE")
	assertContains(t, control, "INDEX_OPTION = 2")
	assertContains(t, control, "CHARACTER_CODE='UTF-8'")
	assertContains(t, control, "LOAD DATA")
	assertContains(t, control, "BADFILE")
	assertContains(t, control, "APPEND")
	assertContains(t, control, "INTO TABLE SYSDBA.BAS_COMPANY")
	assertContains(t, control, "FIELDS TERMINATED BY ','")
	assertContains(t, control, "OPTIONALLY ENCLOSED BY '\"'")
}

func TestBuildCSVControlFileWithNullIf(t *testing.T) {
	file := csvImportFile{
		CSVFile:     "E:/RHPT/TOPI1/BAS_COMPANY.csv",
		Schema:      "SYSDBA",
		Table:       "BAS_COMPANY",
		Columns:     []string{"ID", "COMPANYNO", "COMPANYNAME"},
		ColumnTypes: []string{"CHAR", "CHAR", "CHAR"},
		NullIf:      "NULL",
		Skip:        1,
	}
	options := csvImportOptions{
		Delimiter:     ",",
		CharacterCode: "UTF-8",
		Rows:          50000,
		Direct:        true,
		IndexOption:   2,
		Mode:          "APPEND",
	}

	control := buildCSVControlFile(options, file, "E:/RHPT/TOPI1/BAS_COMPANY.bad")

	// 验证NULLIF处理 - 当NullIf设置为"NULL"时
	assertContains(t, control, "ID CHAR NULLIF ID = 'NULL'")
	assertContains(t, control, "COMPANYNO CHAR NULLIF COMPANYNO = 'NULL'")
	assertContains(t, control, "COMPANYNAME CHAR NULLIF COMPANYNAME = 'NULL'")
}

func TestBuildCSVControlFileWithoutColumnTypes(t *testing.T) {
	file := csvImportFile{
		CSVFile: "E:/RHPT/TOPI1/BAS_COMPANY.csv",
		Schema:  "SYSDBA",
		Table:   "BAS_COMPANY",
		Columns: []string{"ID", "COMPANYNAME"},
		Skip:    1,
	}
	options := csvImportOptions{
		Delimiter:     ",",
		CharacterCode: "UTF-8",
		Rows:          50000,
		Direct:        true,
		IndexOption:   2,
		Mode:          "APPEND",
	}

	control := buildCSVControlFile(options, file, "E:/RHPT/TOPI1/BAS_COMPANY.bad")

	// 验证没有字段类型时，只输出列名
	assertContains(t, control, "ID")
	assertContains(t, control, "COMPANYNAME")
	// 不应该包含字段类型
	assertNotContains(t, control, "ID CHAR")
	assertNotContains(t, control, "COMPANYNAME CHAR")
}

func assertNotContains(t *testing.T, value, want string) {
	t.Helper()
	if contains(value, want) {
		t.Fatalf("expected %q to NOT contain %q", value, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
