package main

import (
	"testing"

	"dm-mcp/tools"
)

func TestBuildOperationToolExposesInternalToolAsTopLevelTool(t *testing.T) {
	info := tools.ToolInfo{
		Name:        "migrate_table_data",
		Description: "migrate data",
		Params:      []string{"source_conn", "target_conn", "table_name", "source_schema", "target_schema", "batch_size"},
	}

	tool := buildOperationTool(info)

	if tool.Name != "migrate_table_data" {
		t.Fatalf("tool.Name = %q, want migrate_table_data", tool.Name)
	}
	for _, param := range info.Params {
		if _, ok := tool.InputSchema.Properties[param]; !ok {
			t.Fatalf("tool schema missing parameter %q", param)
		}
	}
}
