package server

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewHTTPServer 创建 streamable HTTP（2026-07-28 无状态模式）MCP 服务器处理器。
//
// Stateless: true 时，服务器不读写 Mcp-Session-Id 头，每个请求使用临时会话，
// 请求相互独立 —— 与 2026-07-28 规范的无状态核心一致，支持并发与横向扩展。
func NewHTTPServer(name, version string) http.Handler {
	s := newMCPServer(name, version)
	registerTools(s)
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s
	}, &mcp.StreamableHTTPOptions{Stateless: true})
}
