package main

import (
	"dm-mcp/database"
	"dm-mcp/server"
	"flag"
	"log"
	"net/http"
)

// version 发布版本号，CI 构建时可通过 -ldflags "-X main.version=<tag>" 注入。
var version = "2.0.0"

func main() {
	transport := flag.String("transport", "http", "传输方式: http (streamable HTTP, 默认) 或 stdio")
	addr := flag.String("addr", ":8090", "HTTP 监听地址（仅 http 模式）")
	flag.Parse()

	defer database.Close()

	switch *transport {
	case "http":
		handler := server.NewHTTPServer("dm-database-mcp", version)
		log.Printf("达梦数据库 MCP 服务器启动（streamable HTTP, 无状态 2026-07-28），监听 %s", *addr)
		log.Printf("使用 dm_list_tools 查看所有可用工具")
		if err := http.ListenAndServe(*addr, handler); err != nil {
			log.Fatalf("服务器错误: %v", err)
		}
	case "stdio":
		log.Printf("达梦数据库 MCP 服务器启动（stdio）")
		log.Printf("使用 dm_list_tools 查看所有可用工具")
		if err := server.RunStdio("dm-database-mcp", version); err != nil {
			log.Fatalf("服务器错误: %v", err)
		}
	default:
		log.Fatalf("未知传输方式: %s（可选 http|stdio）", *transport)
	}
}
