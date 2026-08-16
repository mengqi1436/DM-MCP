package tools

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// noTruncateTools 豁免结果截断的工具：其数组字段是"结构描述"而非"数据行"，
// 截断会破坏结构完整性（如 describe_table 的列清单、get_table_ddl 的 DDL）。
// 已分页工具（query_paginated）与单行工具（query_one）也不会触发截断。
// dm_list_tools 自带精简/完整两级目录，不参与通用截断。
var noTruncateTools = map[string]bool{
	"describe_table":        true,
	"batch_describe_tables": true,
	"get_table_ddl":         true,
	"query_paginated":       true,
	"query_one":             true,
	"batch_query":           true,
	"dm_list_tools":         true,
}

// Summary 描述一次结果截断的元信息，用于生成人类可读提示。
type Summary struct {
	Field    string   // 被截断的字段名（如 rows）
	Total    int      // 原始条数
	Returned int      // 返回条数
	Fields   []string // 可用字段（来自首条记录）
}

// SummarizeResult 将列表型工具结果截断到 previewLimit 条。
//
// 规则：遍历结果顶层数组字段（任意 slice 类型，如 []map[string]interface{}），
// 超过 previewLimit 时截断并附加标记字段（was_truncated / <field>_total /
// available_fields / summary），保证"截断可见可恢复"：模型可据 summary 判断
// 数据被截断并采取后续动作。结构描述类工具（noTruncateTools）与
// previewLimit<=0 时原样返回。
//
// 返回 (截断后的结果, 截断摘要)；未发生截断时第二个返回值为 nil。
func SummarizeResult(toolName string, v interface{}, previewLimit int) (interface{}, *Summary) {
	if previewLimit <= 0 || noTruncateTools[toolName] {
		return v, nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return v, nil
	}

	out := make(map[string]interface{}, len(m)+2)
	var summary *Summary
	for k, val := range m {
		rv := reflect.ValueOf(val)
		isSlice := rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array)
		if !isSlice || rv.Len() <= previewLimit {
			out[k] = val
			continue
		}
		// 截断为 []interface{}（统一输出形状）
		head := make([]interface{}, 0, previewLimit)
		for i := 0; i < previewLimit; i++ {
			head = append(head, rv.Index(i).Interface())
		}
		out[k] = head
		out[k+"_total"] = rv.Len()
		out["was_truncated"] = true
		if summary == nil {
			summary = &Summary{Field: k, Total: rv.Len(), Returned: previewLimit}
			if f := availableFields(head); len(f) > 0 {
				summary.Fields = f
				out["available_fields"] = f
			}
		}
	}

	if summary == nil {
		return v, nil
	}
	out["summary"] = SummaryText(summary)
	return out, summary
}

// availableFields 从首条记录提取字段名（排序保证输出稳定）。
func availableFields(items []interface{}) []string {
	if len(items) == 0 {
		return nil
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(first))
	for k := range first {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return fields
}

// SummaryText 生成人类可读的截断提示（放进结果的 summary 字段）。
func SummaryText(s *Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结果已截断: 共 %d 条，已返回前 %d 条。", s.Total, s.Returned)
	if len(s.Fields) > 0 {
		fmt.Fprintf(&b, "可用字段: [%s]。", strings.Join(s.Fields, ", "))
	}
	b.WriteString("取全量: 请使用分页查询（如 query_paginated）或增大 limit 参数。")
	return b.String()
}

// EstimateTokenLen 按约 4 字符/token 估算文本长度（供测试断言 token 节省用）。
func EstimateTokenLen(s string) int {
	return len(s) / 4
}
