package main

import (
	"dm-mcp/database"
	_ "dm-mcp/tools"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// version 发布版本号，CI 构建时通过 -ldflags "-X main.version=<tag>" 注入
var version = "2.0.0"

func main() {
	defer database.Close()

	s := server.NewMCPServer(
		"dm-database-mcp",
		version,
		server.WithToolCapabilities(true),
	)

	registerControlTools(s)
	registerOperationTools(s)

	log.Printf("达梦数据库 MCP 服务器启动 (单连接模式)")
	log.Printf("使用 dm_list_tools 查看所有可用工具")
	log.Printf("使用 dm_execute 执行指定工具")

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}

func registerControlTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("dm_list_tools",
			mcp.WithDescription("列出所有可用的达梦数据库操作工具。可按类别筛选：query/dml/ddl/metadata/advanced/admin/import/instance"),
			mcp.WithString("category",
				mcp.Description("筛选类别（可选）: query, dml, ddl, metadata, advanced, admin, import, instance"),
			),
		),
		handleListTools,
	)

	s.AddTool(
		mcp.NewTool("dm_execute",
			mcp.WithDescription("执行指定的达梦数据库操作工具"),
			mcp.WithString("tool_name",
				mcp.Required(),
				mcp.Description("要执行的工具名称"),
			),
			mcp.WithObject("params",
				mcp.Description("工具参数（JSON对象）"),
			),
		),
		handleExecute,
	)
}
