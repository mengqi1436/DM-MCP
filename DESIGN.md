# DM-MCP Design Specification

## 1. Objective

将 dm-mcp 从"单进程多连接"架构重构为"单进程单连接"架构，使其成为无状态、可水平扩展的数据库操作单元。多个 MCP 实例由 mcphub 编排，每个实例绑定一个达梦数据库环境。同时补全达梦官方文档中所有可通过 SQL/CLI 实现的数据库操作功能，并为高频操作提供批量变体。

### 成功标准

- 环境变量从 12 个（PRIMARY_* + SECONDARY_*）简化为 6 个（DM_*）
- 移除 ConnectionManager 多连接抽象，所有工具直接操作唯一连接
- 工具总数从 50 个扩展到 ~80 个，覆盖 DM 官方文档全部可程序化操作
- 批量操作覆盖 DML（batch_update, batch_delete）、DDL（已有）、元数据查询（batch_describe_tables）
- 每个工具无需 `connection` 参数，消除调用歧义

## 2. Product Context

- **用户**：AI Agent（通过 mcphub 调度），非人类直接操作
- **场景**：Agent 需要操作 1~N 个达梦数据库，每个数据库由独立 MCP 实例服务
- **约束**：MCP stdio 协议，单进程单连接，Go 实现
- **竞品参考**：PostgreSQL MCP（单连接模式）、MySQL MCP（单连接模式）

## 3. Architecture

### 3.1 配置模型（单连接）

```
DM_HOST=localhost        # 必填
DM_PORT=5236             # 默认 5236
DM_USER=SYSDBA           # 默认 SYSDBA
DM_PASSWORD=xxx          # 必填
DM_DATABASE=DMDB         # 默认 DMDB
DM_SCHEMA=               # 可选，默认 Schema
```

移除所有 `PRIMARY_*` 和 `SECONDARY_*` 环境变量。移除 `config.LoadPrimaryConfig()`、`config.LoadSecondaryConfig()`、`config.HasSecondaryConfig()`。

### 3.2 连接层

- 移除 `database/manager.go`（ConnectionManager）
- 保留 `database/connection.go` 的单例 `GetDB()` 模式，但改为从新配置加载
- 所有工具通过 `database.GetDB()` 获取唯一连接
- 移除所有工具中的 `connection` 参数和 `queryOnConnection`/`executeOnConnection` 间接层

### 3.3 迁移工具处理

跨库迁移/同步/对比工具（当前 migration 类别 10 个）依赖多连接。处理方案：

- **移除**：`add_connection`, `remove_connection`, `list_connections`（无意义）
- **移除**：`migrate_table_structure`, `migrate_all_tables`, `migrate_table_data`, `migrate_all_data`, `full_migrate`, `sync_table_data`, `compare_table_data`
- **替代**：Agent 层通过 mcphub 编排两个实例完成迁移（从 A 查询 DDL → 在 B 执行 DDL → 从 A 分批查询数据 → 向 B 批量插入）
- **保留辅助**：`export_table_ddl`（导出建表语句）、`export_table_data`（导出为 INSERT 语句或 JSON）供 Agent 编排使用

### 3.4 mcphub 多实例配置示例

```json
{
  "mcpServers": {
    "dm-prod": {
      "command": "D:/MCP/dm-mcp/dm-mcp.exe",
      "env": {
        "DM_HOST": "192.168.1.10",
        "DM_PORT": "5236",
        "DM_USER": "SYSDBA",
        "DM_PASSWORD": "prod_pass",
        "DM_DATABASE": "PROD_DB",
        "DM_SCHEMA": "APP"
      }
    },
    "dm-test": {
      "command": "D:/MCP/dm-mcp/dm-mcp.exe",
      "env": {
        "DM_HOST": "192.168.1.20",
        "DM_PORT": "5237",
        "DM_USER": "SYSDBA",
        "DM_PASSWORD": "test_pass",
        "DM_DATABASE": "TEST_DB"
      }
    }
  }
}
```

## 4. Tool Inventory（完整工具清单）

### 4.1 query — 查询（5 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `query` | 执行 SELECT，返回结果集 | — |
| `query_one` | 查询单条记录 | — |
| `query_paginated` | 分页查询 | — |
| `count` | 统计记录数 | — |
| `batch_query` | 批量执行多条 SELECT，返回各自结果 | 是 |

### 4.2 dml — 数据操作（7 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `insert` | 插入一条 | — |
| `insert_batch` | 批量插入（事务） | 是 |
| `update` | 条件更新 | — |
| `update_batch` | 批量更新（多条不同 WHERE） | 是 |
| `delete` | 条件删除 | — |
| `delete_batch` | 批量删除（多条不同 WHERE） | 是 |
| `merge` | MERGE INTO（UPSERT） | — |

### 4.3 ddl — 数据定义（14 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `create_table` | 创建表 | — |
| `alter_table` | 修改表结构 | — |
| `drop_table` | 删除表 | — |
| `create_index` | 创建索引 | — |
| `drop_index` | 删除索引 | — |
| `execute_ddl` | 执行任意 DDL | — |
| `batch_create_tables` | 批量建表 | 是 |
| `batch_create_indexes` | 批量建索引 | 是 |
| `batch_drop_tables` | 批量删表 | 是 |
| `batch_drop_indexes` | 批量删索引 | 是 |
| `batch_execute_ddl` | 批量执行 DDL 语句 | 是 |
| `create_view` | 创建视图 | — |
| `drop_view` | 删除视图 | — |
| `create_sequence` | 创建序列 | — |

### 4.4 metadata — 元数据（14 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `list_databases` | 列出数据库 | — |
| `list_schemas` | 列出模式 | — |
| `list_tables` | 列出表 | — |
| `list_views` | 列出视图 | — |
| `list_sequences` | 列出序列 | — |
| `list_synonyms` | 列出同义词 | — |
| `list_procedures` | 列出存储过程 | — |
| `list_functions` | 列出函数 | — |
| `list_packages` | 列出包 | — |
| `list_triggers` | 列出触发器 | — |
| `describe_table` | 表结构详情 | — |
| `batch_describe_tables` | 批量获取多表结构 | 是 |
| `list_indexes` | 列出表索引 | — |
| `describe_index` | 索引详情 | — |
| `search_indexes` | 索引检索 | — |
| `get_table_ddl` | 导出建表 DDL | — |
| `list_constraints` | 列出表约束 | — |
| `list_table_partitions` | 列出表分区 | — |

### 4.5 admin — 管理（10 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `database_info` | 数据库基本信息 | — |
| `list_users` | 列出用户 | — |
| `create_user` | 创建用户 | — |
| `drop_user` | 删除用户 | — |
| `grant_privilege` | 授权 | — |
| `revoke_privilege` | 撤权 | — |
| `create_role` | 创建角色 | — |
| `table_statistics` | 表统计信息 | — |
| `list_tablespaces` | 列出表空间 | — |
| `create_tablespace` | 创建表空间 | — |

### 4.6 advanced — 高级（7 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `execute_transaction` | 事务执行多条 SQL | — |
| `call_procedure` | 调用存储过程 | — |
| `call_function` | 调用函数 | — |
| `explain_plan` | 执行计划分析 | — |
| `execute_sql` | 执行任意 SQL（SELECT/DML/DDL 自动判断） | — |
| `batch_execute_sql` | 批量执行任意 SQL | 是 |
| `create_dblink` | 创建 DBLINK | — |

### 4.7 monitoring — 监控诊断（6 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `active_sessions` | 当前活跃会话 | — |
| `lock_info` | 锁信息 | — |
| `slow_queries` | 慢查询（V$SQL_STAT） | — |
| `tablespace_usage` | 表空间使用率 | — |
| `instance_parameters` | 实例参数（dm.ini） | — |
| `session_memory` | 会话内存使用 | — |

### 4.8 backup — 备份恢复（4 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `logical_export` | 逻辑导出（dexp） | — |
| `logical_import` | 逻辑导入（dimp） | — |
| `physical_backup` | 物理备份（dmrman） | — |
| `physical_restore` | 物理恢复（dmrman） | — |

### 4.9 import — 数据导入（2 个）

| 工具 | 说明 | 批量 |
|------|------|------|
| `batch_import_csv` | DMFLDR 批量导入 CSV | 是 |
| `export_table_data` | 导出表数据为 INSERT/JSON | — |

### 4.10 instance — 实例管理（6 个，保留现有）

| 工具 | 说明 |
|------|------|
| `create_database` | dminit 初始化实例 |
| `delete_database` | 删除实例目录 |
| `start_database_service` | 启动服务 |
| `stop_database_service` | 停止服务 |
| `restart_database_service` | 重启服务 |
| `database_service_status` | 服务状态 |

### 4.11 control — 控制工具（2 个，保留）

| 工具 | 说明 |
|------|------|
| `dm_list_tools` | 列出所有工具 |
| `dm_execute` | 执行指定工具 |

**总计：~77 个工具**

## 5. Visual Foundations（接口规范）

- 所有工具返回统一 JSON 结构：`{ "success": bool, ... }`
- 错误返回：MCP error result，包含中文错误信息
- 批量操作返回：`{ "success": bool, "total": N, "ok_count": N, "fail_count": N, "results": [...] }`
- 工具命名：snake_case，动词_名词
- 参数命名：snake_case
- 无 `connection` 参数（单连接，无需指定）

## 6. Accessibility（容错）

- 所有 DML 工具强制 WHERE（delete/update 无 WHERE 拒绝执行）
- DDL 批量操作默认 `atomic=false`（达梦 DDL 隐式提交）
- 备份/恢复/删除操作需要 `confirm=true`
- 密码在日志和输出中脱敏（`******`）
- SQL 注入防护：标识符校验（`dmIdentifierPattern`），值参数化

## 7. Voice & Tone（文档风格）

- 工具描述：中文，一句话说明 + 参数列表
- 错误信息：中文，明确指出缺少哪个参数或哪个操作失败
- README：中文为主，配置示例用 JSON

## 8. Anti-Patterns

- 不在 MCP 内部维护多连接（由 mcphub 多实例解决）
- 不在工具中硬编码 Schema（通过 DM_SCHEMA 环境变量或参数传入）
- 不使用字符串拼接构建 SQL 值（使用参数化查询）
- 不在单连接模式下保留 migration 类别（语义不成立）
- 不添加需要 GUI 交互的功能（如 DM 管理工具图形化操作）

## 9. Decision-Making

当需要新增功能时，判断标准：
1. 达梦官方文档是否支持通过 SQL 或 CLI 工具实现？
2. 是否可以在单连接上完成？
3. 是否有批量需求？如果有，提供 batch_ 变体。
4. 是否需要外部二进制（dexp/dimp/dmrman/dmfldr）？如果是，归入 backup/import 类别。

## 10. Implementation Phases

### Phase 1：架构简化（破坏性变更）
- 重写 `config/config.go`：单配置 `LoadConfig()` 从 `DM_*` 环境变量加载
- 删除 `database/manager.go`
- 重写 `database/connection.go`：移除多连接间接层
- 更新所有工具：移除 `connection` 参数，直接调用 `database.GetDB()`
- 删除 `tools/migration.go`
- 更新 `main.go`：移除 `initPrimaryConnection`/`initSecondaryConnection`
- 更新 README.md

### Phase 2：补全元数据和管理工具
- 新增 `tools/schema_objects.go`：视图、序列、同义词、存储过程、函数、包、触发器的 list/describe
- 新增 `tools/admin_extended.go`：用户管理、角色管理、表空间管理、授权
- 新增 `tools/constraints.go`：约束管理、分区信息

### Phase 3：批量操作和高级功能
- 新增 `batch_update`、`batch_delete`、`batch_query`、`batch_execute_sql`
- 新增 `merge`（MERGE INTO）
- 新增 `execute_sql`（通用 SQL 执行）
- 新增 `get_table_ddl`、`export_table_data`

### Phase 4：监控和备份
- 新增 `tools/monitoring.go`：活跃会话、锁、慢查询、表空间使用率、实例参数
- 新增 `tools/backup.go`：dexp/dimp/dmrman 封装

### Phase 5：文档和测试
- 更新 TOOLS.md 和 README.md
- 补充单元测试
- 更新 mcphub 配置示例
