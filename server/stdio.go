package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio 以 stdio 传输运行 MCP 服务器（本地子进程接入方式，兼容现有编排器）。
func RunStdio(name, version string) error {
	s := newMCPServer(name, version)
	registerTools(s)
	return s.Run(context.Background(), &mcp.StdioTransport{})
}
