# 达梦数据库 MCP v2（高性能版）

基于 **MCP 2026-07-28 无状态规范** 的达梦数据库（DM8）操作工具服务器，Go 实现，性能优先。详细设计见 [DESIGN.md](DESIGN.md)。

## 特性

- **双传输**：streamable HTTP（无状态，2026-07-28 规范）为主 + stdio 兼容，`-transport` 切换
- **82 个工具**：查询、DML、DDL、元数据、管理、监控、备份、CSV 导入、实例管理
- **性能优化**：多行 VALUES 批量插入（10k 行比 v1 快 7.6x）、连接池调优、预编译语句缓存、查询行数上限保护
- **Token 节省（2026-07-28 官方最佳实践）**：
  - **双通道返回**：`content`（紧凑文本）+ `structuredContent`（结构化 JSON，SEP-2106），新客户端直接解析结构化数据
  - **结果摘要化**：大列表结果默认截断到 `DM_LIST_PREVIEW` 条（默认 20），附 `was_truncated`/`_total`/`available_fields`/`summary` 提示，模型可据此分页取全量——实测 1000 行查询省 ~98% token
  - **渐进式工具发现**：`dm_list_tools` 默认返回精简目录（name+category+purpose），`detail=true` 或 `dm_get_tool` 按需加载完整定义——工具列表注入省 ~57%
- **官方 SDK**：`modelcontextprotocol/go-sdk v1.7.0`（原生支持无状态规范）

## 环境要求

- Go 1.21+（**本机需以 `GOARCH=amd64` 编译**，达梦驱动在 32 位 int 下无法编译）
- 达梦数据库 8.x
- 驱动：`gitee.com/chunanyong/dm`（go mod 自动拉取）

## 构建

```powershell
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
tools/                  # 82 个 SQL 工具（registry 模式，协议无关）
DESIGN.md               # 完整设计方案与基准结果
```

## 许可证

MIT License（工具层继承自 v1 `E:\MCP\DM-MCP`）
