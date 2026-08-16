package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestStdioEndToEnd 用官方 SDK client 通过 stdio 连接编译好的 dm-mcp2.exe，
// 验证工具列表与真实数据库调用。需要已 build 的 exe 与数据库凭据。
func TestStdioEndToEnd(t *testing.T) {
	if os.Getenv("DM_PASSWORD") == "" {
		t.Skip("未设置 DM_PASSWORD，跳过 stdio 端到端测试")
	}

	exe := filepath.Join("..", "dm-mcp2.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("未找到 %s，请先 go build", exe)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "v2-test-client", Version: "1.0.0"}, nil)
	transport := &mcp.CommandTransport{Command: exec.Command(exe, "-transport=stdio")}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("stdio 连接失败: %v", err)
	}
	defer session.Close()

	// 工具列表
	listRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools 失败: %v", err)
	}
	if len(listRes.Tools) == 0 {
		t.Fatal("工具列表为空")
	}
	t.Logf("stdio 模式工具数: %d", len(listRes.Tools))

	// 调用 database_info（无需参数，走真实数据库）
	callRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "database_info",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool database_info 失败: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("database_info 返回错误: %s", callRes.Content[0].(*mcp.TextContent).Text)
	}
	text := callRes.Content[0].(*mcp.TextContent).Text
	if !contains(text, "DAMENG") {
		t.Errorf("database_info 结果应包含 DAMENG: %s", text)
	}
	t.Logf("database_info OK: %s", text[:min(120, len(text))])
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
