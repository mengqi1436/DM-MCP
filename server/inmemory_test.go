package server

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestInMemoryCallTool 用 in-memory transport（net.Pipe 纯字节）调用工具，
// 隔离 stdio 传输问题：若此处正常，则问题在 stdio 管道层；否则在数据/序列化层。
func TestInMemoryCallTool(t *testing.T) {
	if os.Getenv("DM_PASSWORD") == "" {
		t.Skip("未设置 DM_PASSWORD，跳过")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	srv := newMCPServer("dm-test", "1.0")
	registerTools(srv)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "database_info",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("in-memory CallTool 失败: %v", err)
	}
	if res.IsError {
		t.Fatalf("返回错误: %s", res.Content[0].(*mcp.TextContent).Text)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	t.Logf("in-memory database_info OK, len=%d, head=%s", len(text), text[:min(80, len(text))])
}
