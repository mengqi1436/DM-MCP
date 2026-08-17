# 达梦数据库 MCP v2 设计方案
## 基于 MCP 2026-07-28 无状态规范的高性能数据库 MCP

- **版本**：3.1.1
- **代码位置**：`E:\MCP\DM-MCP`
- **目标环境**：达梦 DM8，本机 `SYSDBA/<你的密码>@localhost:5236`（数据库 `DAMENG`）
- **状态**：已实现并通过全部验收标准（含真实数据库端到端验证与性能基准）

---

## 1. 概述

本项目是达梦数据库 MCP 服务器的 v2 实现，基于 **MCP 2026-07-28 最新无状态规范**，针对数据库操作场景做**全链路性能优化**。相对 v1（`E:\MCP\DM-MCP`，mark3labs/mcp-go 旧协议 + stdio）的升级点：

| 维度 | v1 | v2 |
|---|---|---|
| 协议规范 | 旧版（initialize + session + stdio） | **2026-07-28 无状态规范** |
| SDK | mark3labs/mcp-go v0.32.0 | **官方 modelcontextprotocol/go-sdk v1.7.0** |
| 传输 | 仅 stdio | **streamable HTTP（无状态）为主 + stdio 兼容** |
| 批量插入 | 逐条 INSERT + JSON 拼接（有注入隐患） | **多行 VALUES + 参数绑定 + 预编译 + 单事务** |
| 连接池 | 固定 10/5，无生命周期管理 | **16/16 + ConnMaxLifetime + ConnMaxIdleTime + 超时控制** |
| Schema | `ALTER SESSION`（仅单连接生效） | **DSN `schema` 属性（驱动自动 set schema，全连接生效）** |
| 查询保护 | 无 | **默认 LIMIT 上限（防大结果集 OOM）** |
| 日志 | `fmt.Printf` 污染 stdout（stdio 下破坏协议帧） | **日志走 stderr，不污染协议流** |

---

## 2. 环境事实（探索确认）

| 项 | 值 |
|---|---|
| 达梦安装 | `E:\DB\dmdbms`（DM8，dexp/dimp/dmfldr/dmrman/disql 全套） |
| 实例 | `DmServiceDMSERVER` 运行中，监听 `5236`；实例名 `DMSERVER` |
| 数据库 | `DAMENG`（**v1 默认 `DMDB` 为错误值，v2 修正**） |
| Go | 1.26.5（windows/386；**v2 以 `GOARCH=amd64 CGO_ENABLED=0` 交叉编译**，因达梦驱动在 32 位 int 下 `0xffffffff` 溢出） |
| 官方 go-sdk | `modelcontextprotocol/go-sdk v1.7.0`（原生支持无状态：`StreamableHTTPOptions.Stateless`） |
| 达梦驱动 | `gitee.com/chunanyong/dm v1.8.23`（标准 database/sql；DSN 支持 `schema`、`rowPrefetch` 等属性） |

### 关键驱动事实（实现中发现）
- 达梦**一个实例即一个数据库**，驱动 DSN 的 **URL path 会被当作 schema** 执行 `set schema` → v2 的 DSN 不带 path。
- 驱动 DSN 的 query 属性 `schema=xxx` 会在**每连接建立时自动执行 `set schema`**（源码 `n.go:975`），对连接池所有连接生效——这是 v2 连接层 schema 方案的核心依据。
- 驱动默认返回**大写列名**（`CNT` 而非 `cnt`）→ 修复了 v1 遗留的 `count` 工具取 `results[0]["cnt"]` 为 null 的 bug。

---

## 3. MCP 2026-07-28 规范要点（方案依据）

来源：[官方 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)、[mcp-uplift](https://github.com/MohibShaikh/mcp-uplift)、[AAIF 迁移分析](https://aaif.io/blog/mcp-2026-07-28-whats-changing-and-how-to-migrate)、[appwrite 解读](https://appwrite.io/blog/post/mcp-goes-stateless-in-the-2026-07-28-specification)。

| 变化 | 影响 |
|---|---|
| **无状态核心**：移除 `initialize` 握手与 `notifications/initialized` | 客户端直接发请求，无"连接-协商-对话"开销 |
| **移除 Sessions**：`Mcp-Session-Id` 不再必需 | 每个请求独立；官方 SDK `Stateless:true` 时 GET/DELETE 返回 405 |
| **移除 server-initiated requests** | server→client 请求被拒绝；roots/sampling/logging 进入 ≥12 个月弃用窗口（SEP-2577） |
| **streamable HTTP 为主传输**，支持 JSON-RPC batch | 单请求多调用、并发、横向扩展 |
| 官方 go-sdk v1.7.0 配套此规范 | `serveStateless`/`serveStateful` 双模式；stdio 传输仍要求 legacy initialize（实测确认） |

**对数据库 MCP 的意义**：MCP 层无状态，数据库长连接由服务器内连接池持有，两者解耦 → 服务器可多实例横向扩展、连接池共享、并发请求天然支持。

---

## 4. 技术选型

| 项 | 选择 | 理由 |
|---|---|---|
| 语言 | Go | 编译型、并发强；复用 v1 工具层 |
| SDK | 官方 `go-sdk v1.7.0` | 唯一原生支持 2026-07-28 无状态的官方 Go SDK（mark3labs 仍在演进） |
| 传输 | streamable HTTP（无状态）为主 + stdio 兼容 | 2026-07-28 推荐形态 + 本地编排兼容 |
| 驱动 | `gitee.com/chunanyong/dm v1.8.23` | 标准 database/sql、社区最成熟；DSN 属性丰富 |

---

## 5. 架构

```
┌─────────────── MCP 客户端（Agent / 编排器）───────────────┐
│  streamable HTTP (无状态, 2026-07-28)      stdio         │
└──────────────────────┬───────────────────────────────────┘
                       ▼
┌─────────────── dm-mcp2 (E:\MCP\DM-mcp-2.0) ──────────────┐
│  main.go: -transport=http|stdio                            │
│  server/http.go  : NewStreamableHTTPHandler{Stateless:true}│
│  server/stdio.go : StdioTransport                          │
│  server/operation_tools.go: v1 元数据 → 官方 SDK Tool(schema)│
│  tools/*.go     : 83 个 SQL 工具（registry 模式，协议无关） │
│  database/connection.go: 调优连接池 + 批量助手 + 超时      │
│  config/config.go: DM_* 环境变量（含连接池/执行参数）       │
└──────────────────────────┬────────────────────────────────┘
                           ▼
             达梦 DB (DAMENG @ localhost:5236, SYSDBA)
```

## 6. 分层性能优化策略（"性能最佳"核心）

### 6.1 协议/服务层
- 无状态 streamable HTTP：并发请求、无粘性会话、多实例横向扩展
- 工具 schema 精简：由 v1 参数元数据自动生成 JSON Schema 2020-12，`required` 从描述 `(必填)` 提取
- 日志全部走 stderr，**绝不污染 stdout 协议流**（修复 v1 的 stdio 帧污染缺陷）

### 6.2 连接层（`database/connection.go`）

| 参数 | 默认 | 说明 |
|---|---|---|
| `DM_MAX_OPEN_CONNS` | 16 | 并发上限 |
| `DM_MAX_IDLE_CONNS` | 16 | 保持热连接，避免重复握手 |
| `DM_CONN_MAX_LIFETIME` | 30m | 避免陈旧连接 |
| `DM_CONN_MAX_IDLE_TIME` | 5m | 回收空闲连接 |
| `DM_QUERY_TIMEOUT` | 30s | 每请求超时（防挂死） |
| `DM_BATCH_SIZE` | 500 | 批量写入分块 |
| `DM_QUERY_LIMIT` | 1000 | 查询默认行数上限 |

- Schema 通过 DSN `schema` 属性传递，驱动在每连接建立时自动 `set schema`（全连接池生效，替代 v1 的 `ALTER SESSION` 单连接方案）
- `DM_DRIVER_PARAMS` 可透传更多驱动属性（如 `rowPrefetch=100&stmtPoolMaxSize=50`）

### 6.3 执行层（工具内部）
- **批量 DML 用多行 VALUES 参数绑定**（`ExecuteBatchInsert`）：`INSERT INTO t (c1,c2) VALUES (:1,:2),(:3,:4)...`，按 `DM_BATCH_SIZE` 分块、单事务、失败返回成功批数
- **插入值全部参数绑定**，杜绝 v1 的 JSON 字符串拼接（既有注入隐患又慢）
- 查询默认行数上限：`query` 自动追加 `LIMIT <DM_QUERY_LIMIT>`（可被显式 LIMIT 或 `limit` 参数覆盖）
- `batch_import_csv`：直接调用达梦官方高速加载器 `dmfldr.exe`（`DM_FLDR_PATH`）

### 6.4 工具集设计
复用 v1 全部 82 个工具（MIT 许可，`tools/` 逻辑协议无关）。**高频核心集（性能优先）**：`query`、`query_one`、`query_paginated`、`count`、`batch_query`、`insert`、`insert_batch`、`update`、`delete`、`execute_sql`、`execute_transaction`、`list_tables`、`describe_table`、`get_table_ddl`、`batch_import_csv` —— 全部走 6.2/6.3 优化路径。

---

## 7. 完整工具清单（83 个）

- **control(3)**：dm_list_tools、dm_get_tool、dm_execute
- **query(5)**：query、query_one、query_paginated、count、batch_query
- **dml(7)**：insert、insert_batch、update、update_batch、delete、delete_batch、merge
- **ddl(14)**：create_table、alter_table、drop_table、create_index、drop_index、execute_ddl、batch_create_tables、batch_create_indexes、batch_drop_tables、batch_drop_indexes、create_view、drop_view、create_sequence、drop_sequence
- **metadata(18)**：list_databases、list_schemas、list_tables、list_views、list_sequences、list_synonyms、list_procedures、list_functions、list_packages、list_triggers、describe_table、batch_describe_tables、list_indexes、describe_index、search_indexes、get_table_ddl、list_constraints、list_table_partitions
- **admin(12)**：database_info、list_users、create_user、drop_user、grant_privilege、revoke_privilege、create_role、drop_role、list_roles、table_statistics、list_tablespaces、create_tablespace
- **advanced(6)**：execute_transaction、call_procedure、call_function、explain_plan、execute_sql、batch_execute_sql
- **monitoring(6)**：active_sessions、lock_info、slow_queries、tablespace_usage、instance_parameters、session_memory
- **backup(4)**：logical_export、logical_import、physical_backup、physical_restore
- **import(2)**：batch_import_csv、export_table_data
- **instance(6)**：create_database、delete_database、start_database_service、stop_database_service、restart_database_service、database_service_status

---

## 8. 配置项（本机示例）

```ini
DM_HOST=localhost
DM_PORT=5236
DM_USER=SYSDBA
DM_PASSWORD=<你的密码>
DM_DATABASE=DAMENG
DM_SCHEMA=                  # 可选：驱动自动 set schema
DM_MAX_OPEN_CONNS=16
DM_MAX_IDLE_CONNS=16
DM_CONN_MAX_LIFETIME=30m
DM_CONN_MAX_IDLE_TIME=5m
DM_QUERY_TIMEOUT=30s
DM_BATCH_SIZE=500
DM_QUERY_LIMIT=1000
DM_DRIVER_PARAMS=           # 可选：rowPrefetch=100&stmtPoolMaxSize=50
DM_FLDR_PATH=E:\DB\dmdbms\bin\dmfldr.exe
DM_EXP_PATH=E:\DB\dmdbms\bin\dexp.exe
DM_IMP_PATH=E:\DB\dmdbms\bin\dimp.exe
DM_RMAN_PATH=E:\DB\dmdbms\bin\dmrman.exe
```

---

## 9. 部署形态

1. **本地 HTTP 服务**：`dm-mcp2.exe -transport=http -addr=:8090`，供支持 2026-07-28 streamable HTTP 的客户端连接（官方 SDK client / Codex 等）。
2. **stdio 本地进程**：`dm-mcp2.exe -transport=stdio`，供本地编排器以子进程方式拉起（与 v1 相同接入方式）。
3. **横向扩展**：无状态 HTTP 多副本 + 负载均衡（nginx），连接池各自管理达梦连接。

---

## 10. 安全与容错

- 密码仅从环境变量读取；日志/错误输出不打印密码
- 所有 DML 强制 WHERE（delete/update 无 WHERE 拒绝执行）
- SQL 注入防护：标识符白名单校验 + 全部参数化绑定（`insert_batch` 已消除 v1 的字符串拼接路径）
- 备份/恢复/删除类操作需 `confirm=true`
- 每请求超时（`DM_QUERY_TIMEOUT`）防止慢查询挂死连接池
- 大结果集默认 LIMIT 上限防 OOM

---

## 11. 边界情况与失败模式

| 场景 | 处理 |
|---|---|
| 无状态下的事务 | `execute_transaction` 为单请求内多语句事务；跨请求事务不支持（2026-07-28 无状态约束） |
| 大结果集 | 默认 limit 1000（`DM_QUERY_LIMIT`），超出提示分页 |
| 批量写入失败 | 分块执行，失败返回成功批数 + 错误详情并回滚当前事务 |
| 达梦 DDL 隐式提交 | `atomic` 参数仅对 DML 语义成立，DDL 批量推荐 `atomic=false` |
| stdio 协议流污染 | 日志强制 stderr；`config` 警告不再输出到 stdout（修复 v1 缺陷） |
| 驱动列名 | 达梦返回大写列名，`count` 等工具大小写不敏感取值 |
| 32 位 Go | 驱动 `0xffffffff` 溢出 → 必须以 `GOARCH=amd64` 编译 |

---

## 12. 基准测试结果（实测，本机 i7-12700K / 达梦 DAMENG）

环境：`go test -bench=. -benchtime=1x dm-mcp2/database`，DM_BATCH_SIZE=500。

| 场景 | v2 多行 VALUES | v1 旧式逐条 | **提升** |
|---|---|---|---|
| 批量插入 1000 行 | 61.5 ms | 145.9 ms | **2.4x** |
| 批量插入 10,000 行 | 518 ms | 3,935 ms | **7.6x** |
| 批量插入 100,000 行 | 4.8 s | — | — |
| 全表扫描 100k 行（含序列化） | 451 ms | — | — |
| 分页查询 1000 行 | 4.3 ms | — | — |

批量写入吞吐随数据量增长呈线性扩展（约 18k~21k rows/s），且提升幅度随行数增大更显著（预编译 + 减少往返次数）。

复现命令：

```powershell
$env:DM_HOST="localhost"; $env:DM_PORT="5236"; $env:DM_USER="SYSDBA"; $env:DM_PASSWORD="..."; $env:DM_DATABASE="DAMENG"
go test -bench=. -benchtime=1x dm-mcp2/database
```

---

## 13. 实现与验证记录

### 实现结构

```
DM-mcp-2.0/
├── DESIGN.md                  # 本方案
├── README.md                  # 安装/配置/用法
├── go.mod / go.sum            # go-sdk v1.7.0 + dm v1.8.23
├── main.go                    # -transport=http|stdio（默认 http，-addr=:8090）
├── config/config.go           # DM_* 配置（连接池/执行参数/驱动透传）
├── config/config_test.go      # 配置与 DSN 单元测试
├── database/
│   ├── connection.go          # 调优连接池 + ExecuteBatchInsert 批量助手
│   ├── connection_test.go     # 连接 + 批量插入集成测试（需凭据）
│   └── benchmark_test.go      # 多行 VALUES vs 旧式逐条基准
├── server/
│   ├── http.go                # StreamableHTTPHandler{Stateless:true}
│   ├── stdio.go               # StdioTransport
│   ├── operation_tools.go     # 工具注册 + schema 生成 + 适配器
│   ├── operation_tools_test.go# schema/参数解析单元测试
│   ├── inmemory_test.go       # in-memory 端到端隔离测试
│   └── stdio_test.go          # 官方 client 连子进程端到端测试
└── tools/                     # 复制自 v1（MIT）并适配
```

### 验证结果

| 验收项 | 结果 |
|---|---|
| `go build -o dm-mcp2.exe`（amd64） | ✅ 16.6 MB |
| `go test ./...`（单元 + 集成） | ✅ 全部通过 |
| HTTP 无状态 `tools/list` | ✅ 82 工具 |
| HTTP 无状态 `tools/call`（database_info/query/count/create_table/insert_batch/drop_table） | ✅ 真实数据 |
| 无状态特性（无 session-id 独立请求） | ✅ |
| stdio（官方 SDK client 连子进程） | ✅ 82 工具 + database_info |
| `insert_batch` 1000 行端到端 | ✅ `inserted:1000`，数据库层确认 1000 行 |
| 性能基准 | ✅ 10k 行批量插入 7.6x 提升 |

### 实现中发现并修复的问题

1. **DSN path 被驱动当 schema**（v1 的 `dm://.../DAMENG` 报"无效的模式名"）→ v2 DSN 不带 path，schema 走 query 属性。
2. **32 位 Go 下驱动编译失败**（`0xffffffff` 溢出 int）→ 以 `GOARCH=amd64 CGO_ENABLED=0` 交叉编译。
3. **config 日志污染 stdout 破坏 stdio 协议帧**（`invalid character 'é'` 根因）→ 日志改走 stderr。
4. **v1 遗留 bug**：`insert_batch` 用 JSON 字符串拼接（注入隐患 + 慢）→ 多行 VALUES 参数绑定；`count` 取小写 `cnt` 列（达梦返回大写 `CNT`）→ 大小写不敏感。

---

## 14. 迁移指南（v1 → v2）

1. **替换二进制**：`dm-mcp2.exe` 替代 `dm-mcp.exe`（注意 amd64 架构）。
2. **环境变量**：
   - 沿用全部 `DM_*` 变量；**修正 `DM_DATABASE=DAMENG`**（v1 默认 `DMDB` 是错的）
   - 可选新增：`DM_MAX_OPEN_CONNS`、`DM_BATCH_SIZE`、`DM_QUERY_LIMIT` 等性能参数
3. **传输**：
   - 本地编排（mcphub 等）：`-transport=stdio`（与 v1 接入方式一致）
   - 需要并发/远程/横向扩展：`-transport=http -addr=:8090`（2026-07-28 streamable HTTP，无状态）
4. **工具**：82 个工具名称与参数完全兼容 v1，Agent 侧无需改动调用。

---

## 15. 后续优化方向

- 达梦官方 `dm-go-driver`（`E:\DB\dmdbms\drivers\go\dm-go-driver.zip`）作为驱动对照基准评估
- 服务端分页游标（`keyset pagination`）替代 `LIMIT/OFFSET`（深分页性能）
- JSON-RPC batch 客户端适配（一次请求多查询）
- HTTP 模式接入 OAuth 2.1（2026-07-28 规范配套）
- 连接池动态扩缩容（按并发负载自适应 `MaxOpenConns`）

---

## 16. Token 节省改造（已实现）

基于 MCP 官方最佳实践（context7 检索 + go-sdk v1.7.0 源码核实）的分类分级返回方案：

| 维度 | 改造 | 效果（实测） |
|---|---|---|
| 结果分级 | `jsonResult` 双通道：`content` 单行 JSON + `structuredContent`（SEP-2106）；大列表按 `DM_LIST_PREVIEW`（默认 20）截断，附 `was_truncated`/`_total`/`available_fields`/`summary` | 1000 行查询 ~39.7k → ~0.9k tokens（**省 97.8%**） |
| 工具定义分级 | `dm_list_tools` 默认精简目录（name+category+purpose）；`detail=true` 返回完整；新增 `dm_get_tool` 按需加载单工具 | 工具目录注入 7.3k → 3.1k tokens（**省 57%**） |
| 结构豁免 | `noTruncateTools`：describe_table/batch_describe_tables/get_table_ddl/query_paginated/query_one/batch_query/dm_list_tools 不截断（结构完整性优先） | 宽表列清单、DDL 不被误截断 |
| 配置 | `DM_LIST_PREVIEW`（默认 20） | 按需调大/调小 |

实现位置：`server/operation_tools.go`（`jsonResult` 双通道 / `dm_get_tool`）、`tools/response.go`（`SummarizeResult` 纯函数）、`config/config.go`。测试：`tools/response_test.go`、`server/operation_tools_test.go` 扩展。
