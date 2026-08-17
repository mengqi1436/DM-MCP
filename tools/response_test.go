package tools

import (
	"strings"
	"testing"
)

// TestSummarizeResultTruncatesLargeList 用真实数据类型 []map[string]interface{}
// （queryDB 返回类型）验证截断逻辑。
func TestSummarizeResultTruncatesLargeList(t *testing.T) {
	items := make([]map[string]interface{}, 0, 100)
	for i := 0; i < 100; i++ {
		items = append(items, map[string]interface{}{"ID": i, "NAME": "n"})
	}
	in := map[string]interface{}{"rows": items, "count": 100}

	out, summary := SummarizeResult("query", in, 20)
	if summary == nil {
		t.Fatal("应返回截断摘要")
	}
	m := out.(map[string]interface{})
	rows, ok := m["rows"].([]interface{})
	if !ok {
		t.Fatalf("rows 应为数组, got %T", m["rows"])
	}
	if len(rows) != 20 {
		t.Errorf("rows 应截断到 20 条, got %d", len(rows))
	}
	if m["was_truncated"] != true {
		t.Error("应标记 was_truncated")
	}
	if m["rows_total"] != 100 {
		t.Errorf("rows_total 应为 100, got %v", m["rows_total"])
	}
	af, ok := m["available_fields"].([]string)
	if !ok || len(af) != 2 {
		t.Errorf("available_fields 应为 2 个字段, got %v", m["available_fields"])
	}
	sm, ok := m["summary"].(string)
	if !ok || !strings.Contains(sm, "共 100 条") || !strings.Contains(sm, "前 20 条") {
		t.Errorf("summary 应包含条数信息, got %q", sm)
	}
}

func TestSummarizeResultKeepsSmallList(t *testing.T) {
	items := []map[string]interface{}{{"ID": 1}}
	in := map[string]interface{}{"rows": items, "count": 1}
	out, summary := SummarizeResult("query", in, 20)
	if summary != nil {
		t.Error("小列表不应返回截断摘要")
	}
	if _, ok := out.(map[string]interface{})["was_truncated"]; ok {
		t.Error("小列表不应标记 was_truncated")
	}
}

// TestSummarizeResultExactLimit 覆盖 rv.Len() <= previewLimit 的精确边界：
// 列表长度恰好等于阈值时不截断（变异体 CONDITIONALS_BOUNDARY）。
func TestSummarizeResultExactLimit(t *testing.T) {
	items := make([]map[string]interface{}, 0, 20)
	for i := 0; i < 20; i++ {
		items = append(items, map[string]interface{}{"ID": i})
	}
	in := map[string]interface{}{"rows": items, "count": 20}
	out, summary := SummarizeResult("query", in, 20)
	if summary != nil {
		t.Fatal("长度等于阈值时不应截断")
	}
	if _, ok := out.(map[string]interface{})["was_truncated"]; ok {
		t.Error("长度等于阈值时不应标记 was_truncated")
	}
}

// TestSummarizeResultEmptyFields 覆盖 availableFields 返回空（首条非 map）
// 时 len(f) > 0 分支为假的情况（变异体 CONDITIONALS_BOUNDARY/NEGATION）。
func TestSummarizeResultEmptyFields(t *testing.T) {
	items := make([]interface{}, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items, i) // 非 map 元素 → 无可用字段
	}
	in := map[string]interface{}{"rows": items}
	out, summary := SummarizeResult("query", in, 20)
	if summary == nil {
		t.Fatal("应截断")
	}
	m := out.(map[string]interface{})
	if _, ok := m["available_fields"]; ok {
		t.Error("首条非 map 时不应设置 available_fields")
	}
	if summary.Fields != nil {
		t.Error("summary.Fields 应为 nil")
	}
}

// TestSummaryTextWithoutFields 覆盖 SummaryText 无字段时的分支
// （变异体 CONDITIONALS_BOUNDARY 在 len(s.Fields) > 0）。
func TestSummaryTextWithoutFields(t *testing.T) {
	s := &Summary{Field: "rows", Total: 100, Returned: 20}
	text := SummaryText(s)
	if !strings.Contains(text, "共 100 条") {
		t.Errorf("应包含条数: %s", text)
	}
	if strings.Contains(text, "可用字段") {
		t.Errorf("无字段时不应出现可用字段提示: %s", text)
	}
	if !strings.Contains(text, "取全量") {
		t.Errorf("应包含取全量提示: %s", text)
	}
}

func TestSummarizeResultSkipsStructuralTools(t *testing.T) {
	cols := make([]map[string]interface{}, 0, 50)
	for i := 0; i < 50; i++ {
		cols = append(cols, map[string]interface{}{"COLUMN_NAME": "c"})
	}
	in := map[string]interface{}{"table_name": "T", "columns": cols, "count": 50}
	_, summary := SummarizeResult("describe_table", in, 20)
	if summary != nil {
		t.Error("describe_table 为结构描述工具，不应截断")
	}
}

func TestSummarizeResultSkipsListTools(t *testing.T) {
	items := make([]map[string]interface{}, 100)
	in := map[string]interface{}{"tools": items, "total": 100}
	_, summary := SummarizeResult("dm_list_tools", in, 20)
	if summary != nil {
		t.Error("dm_list_tools 自带分级，不应被通用截断")
	}
}

func TestSummarizeResultZeroPreviewDisabled(t *testing.T) {
	items := make([]map[string]interface{}, 100)
	in := map[string]interface{}{"rows": items}
	_, summary := SummarizeResult("query", in, 0)
	if summary != nil {
		t.Error("previewLimit<=0 时不应截断")
	}
}

func TestSummarizeResultNonMapPassthrough(t *testing.T) {
	in := []interface{}{1, 2, 3}
	out, summary := SummarizeResult("query", in, 20)
	if summary != nil || out == nil {
		t.Error("非 map 输入应原样返回")
	}
}

// estimateTokenLen 按约 4 字符/token 估算文本长度（测试辅助）。
func estimateTokenLen(s string) int {
	return len(s) / 4
}

func TestEstimateTokenLen(t *testing.T) {
	if estimateTokenLen(strings.Repeat("a", 400)) != 100 {
		t.Error("估算 400 字符应约 100 token")
	}
}
