package main

import (
	"dm-mcp/database"
	_ "dm-mcp/tools" // 触发工具注册
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 初始化数据库连接
	if _, err := database.GetDB(); err != nil {
		log.Printf("警告: 数据库连接失败: %v", err)
	}
	defer database.Close()

	// 创建MCP服务器
	s := server.NewMCPServer(
		"dm-database-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// 注册控制工具
	registerControlTools(s)

	log.Println("达梦数据库 MCP 服务器启动")
	log.Println("已注册 2 个控制工具，35 个操作工具")
	log.Println("使用 dm_list_tools 查看所有可用工具")
	log.Println("使用 dm_execute 执行指定工具")

	// 运行服务器
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}

// registerControlTools 注册控制工具
func registerControlTools(s *server.MCPServer) {
	// dm_list_tools - 列出所有可用工具
	s.AddTool(
		mcp.NewTool("dm_list_tools",
			mcp.WithDescription("列出所有可用的达梦数据库操作工具。可按类别筛选：query/dml/ddl/metadata/advanced/admin/import"),
			mcp.WithString("category",
				mcp.Description("筛选类别（可选）: query, dml, ddl, metadata, advanced, admin, import"),
			),
		),
		handleListTools,
	)

	// dm_execute - 执行指定工具
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
