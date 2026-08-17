# 达梦数据库 MCP 服务器（dm-mcp）

基于 **MCP 2026-07-28 无状态规范** 的达梦数据库（DM8）操作工具服务器，Go 实现，性能优先。已发布到 [npm](https://www.npmjs.com/package/dm-mcp) 与 [GitHub Releases](https://github.com/mengqi1436/DM-MCP/releases)。

## 特性

- **双传输**：streamable HTTP（无状态，2026-07-28 规范）为主 + stdio 兼容，`-transport` 切换
- **83 个工具**：查询、DML、DDL、元数据、管理、监控、备份、CSV 导入、实例管理
- **性能优化**：多行 VALUES 批量插入（10k 行比 v1 快 7.6x）、连接池调优、查询行数上限保护
- **Token 节省（2026-07-28 官方最佳实践）**：
  - **双通道返回**：`content`（紧凑文本）+ `structuredContent`（结构化 JSON，SEP-2106），新客户端直接解析结构化数据
  - **结果摘要化**：大列表结果默认截断到 `DM_LIST_PREVIEW` 条（默认 20），附 `was_truncated`/`_total`/`available_fields`/`summary` 提示，模型可据此分页取全量——实测 1000 行查询省 ~98% token
  - **渐进式工具发现**：`dm_list_tools` 默认返回精简目录（name+category+purpose），`detail=true` 或 `dm_get_tool` 按需加载完整定义——工具列表注入省 ~57%
- **官方 SDK**：`modelcontextprotocol/go-sdk v1.7.0`（原生支持无状态规范）

## 环境要求

- Go 1.26.5+（源码构建；**需以 `GOARCH=amd64` 编译**，达梦驱动在 32 位 int 下无法编译）
- 达梦数据库 8.x
- 驱动：`gitee.com/chunanyong/dm`（go mod 自动拉取）

## 安装

### 方式一：npm（推荐，免编译）

```powershell
npm install -g dm-mcp
```

安装后提供 `dm-mcp` 命令（内置 win32/linux/darwin × x64/arm64 预编译二进制，以 stdio 模式启动）：

```powershell
dm-mcp
```

### 方式二：源码构建

```powershell
git clone https://github.com/mengqi1436/DM-MCP.git
cd DM-MCP
$env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o dm-mcp2.exe .
```

## 配置（环境变量）

| 变量 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `DM_HOST` | 否 | `localhost` | 主机 |
| `DM_PORT` | 否 | `5236` | 端口 |
| `DM_USER` | 否 | `SYSDBA` | 用户 |
| `DM_PASSWORD` | **是** | — | 密码 |
| `DM_DATABASE` | 否 | `DAMENG` | 数据库名（展示用） |
| `DM_SCHEMA` | 否 | — | 默认 Schema（驱动自动 set schema） |
| `DM_MAX_OPEN_CONNS` | 否 | `16` | 连接池上限 |
| `DM_MAX_IDLE_CONNS` | 否 | `16` | 空闲连接数 |
| `DM_CONN_MAX_LIFETIME` | 否 | `30m` | 连接存活时长 |
| `DM_CONN_MAX_IDLE_TIME` | 否 | `5m` | 空闲回收时长 |
| `DM_QUERY_TIMEOUT` | 否 | `30s` | 单请求超时 |
| `DM_BATCH_SIZE` | 否 | `500` | 批量写入分块 |
| `DM_QUERY_LIMIT` | 否 | `1000` | 查询默认行数上限 |
| `DM_LIST_PREVIEW` | 否 | `20` | 列表结果默认预览条数（超限截断并附摘要提示，防上下文膨胀） |
| `DM_DRIVER_PARAMS` | 否 | — | 驱动属性透传（如 `rowPrefetch=100`） |
| `DM_FLDR_PATH` | 否 | `dmfldr` | dmfldr 路径（CSV 导入） |
| `DM_EXP_PATH` | 否 | `dexp` | dexp 路径（逻辑导出） |
| `DM_IMP_PATH` | 否 | `dimp` | dimp 路径（逻辑导入） |
| `DM_RMAN_PATH` | 否 | `dmrman` | dmrman 路径（物理备份） |

本机示例：

```powershell
$env:DM_HOST="localhost"; $env:DM_PORT="5236"; $env:DM_USER="SYSDBA"
$env:DM_PASSWORD="你的密码"; $env:DM_DATABASE="DAMENG"
```

## 运行

### HTTP 模式（默认，2026-07-28 无状态）

```powershell
dm-mcp2.exe -transport=http -addr=:8090
```

调用示例（JSON-RPC over HTTP，无需 session）：

```bash
curl -X POST http://localhost:8090/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"database_info","arguments":{}}}'
```

### stdio 模式（本地编排器子进程接入）

```powershell
dm-mcp2.exe -transport=stdio
```

## 工具集（83 个）

| 类别 | 工具 |
|---|---|
| **control(3)** | `dm_list_tools`、`dm_get_tool`、`dm_execute` |
| **query(5)** | `query`、`query_one`、`query_paginated`、`count`、`batch_query` |
| **dml(7)** | `insert`、`insert_batch`、`update`、`update_batch`、`delete`、`delete_batch`、`merge` |
| **ddl(14)** | `create_table`、`alter_table`、`drop_table`、`create_index`、`drop_index`、`execute_ddl`、`batch_create_tables`、`batch_create_indexes`、`batch_drop_tables`、`batch_drop_indexes`、`create_view`、`drop_view`、`create_sequence`、`drop_sequence` |
| **metadata(18)** | `list_databases`、`list_schemas`、`list_tables`、`list_views`、`list_sequences`、`list_synonyms`、`list_procedures`、`list_functions`、`list_packages`、`list_triggers`、`describe_table`、`batch_describe_tables`、`list_indexes`、`describe_index`、`search_indexes`、`get_table_ddl`、`list_constraints`、`list_table_partitions` |
| **admin(12)** | `database_info`、`list_users`、`create_user`、`drop_user`、`grant_privilege`、`revoke_privilege`、`create_role`、`drop_role`、`list_roles`、`table_statistics`、`list_tablespaces`、`create_tablespace` |
| **advanced(6)** | `execute_transaction`、`call_procedure`、`call_function`、`explain_plan`、`execute_sql`、`batch_execute_sql` |
| **monitoring(6)** | `active_sessions`、`lock_info`、`slow_queries`、`tablespace_usage`、`instance_parameters`、`session_memory` |
| **backup(4)** | `logical_export`、`logical_import`、`physical_backup`、`physical_restore` |
| **import(2)** | `batch_import_csv`、`export_table_data` |
| **instance(6)** | `create_database`、`delete_database`、`start_database_service`、`stop_database_service`、`restart_database_service`、`database_service_status` |

## 测试

```powershell
# 单元测试（无需数据库）
go test ./...

# 集成 + 端到端测试（需设置 DM_PASSWORD 等环境变量）
go test -v ./database/ ./server/

# 性能基准（连接真实库）
go test -bench=. -benchtime=1x dm-mcp2/database
```

## 项目结构

```
main.go                 # 入口（-transport flag）
config/                 # DM_* 配置（连接池/执行参数）
database/               # 调优连接池 + 批量助手 + 基准测试
server/                 # 官方 SDK 协议层（HTTP 无状态 / stdio）
tools/                  # 83 个 SQL 工具（registry 模式，协议无关）
npm/                    # npm 发布包装（cli.js 启动器 + 预编译二进制）
.github/workflows/      # CI 与 Release（构建 5 平台二进制并发布 npm）
DESIGN.md               # 设计方案与基准结果
```

## 许可证

MIT License
