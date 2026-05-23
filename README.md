# 达梦数据库 MCP 工具

一个基于 Go 语言和 MCP（Model Context Protocol）开发的达梦数据库操作工具服务器。

## 功能特性

### 工具发现机制

服务器提供 **2 个控制工具** 和 **35 个操作工具**：

#### 控制工具
- `dm_list_tools` - 列出所有可用工具（可按类别筛选）
- `dm_execute` - 执行指定的操作工具

#### 操作工具类别（35 个）

##### query — 查询工具（4 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `query` | 执行 SQL SELECT 查询，返回结果集 | `sql` | `params` |
| `query_one` | 执行查询，只返回第一条记录（适合按 ID 查单条） | `sql` | `params` |
| `query_paginated` | 分页查询，自动拼接 LIMIT/OFFSET | `sql` | `page`（默认 1）、`page_size`（默认 20） |
| `count` | 统计表记录数，支持 WHERE 条件 | `table` | `where` |

##### dml — 数据操作工具（4 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `insert` | 插入一条数据，自动生成参数化 SQL | `table`、`data`（字段名:值） | — |
| `insert_batch` | 批量插入多条数据，使用事务保证原子性 | `table`、`rows`（数据数组） | — |
| `update` | 按条件更新数据（必须带 WHERE 防止全表更新） | `table`、`data`、`where` | — |
| `delete` | 按条件删除数据（必须带 WHERE 防止全表删除） | `table`、`where` | — |

##### ddl — 数据定义工具（10 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `create_table` | 创建表，支持列类型、主键、非空、默认值定义 | `table_name`、`columns` | — |
| `alter_table` | 修改表结构（ADD/MODIFY/DROP 列） | `table_name`、`operation`、`column` | `type`（ADD/MODIFY 时需要） |
| `drop_table` | 删除表 | `table_name` | `if_exists` |
| `create_index` | 创建索引 | `index_name`、`table_name`、`columns` | `unique` |
| `drop_index` | 删除索引，支持 `SCHEMA.INDEX` 全名 | `index_name` | `schema`、`if_exists` |
| `execute_ddl` | 执行任意 DDL 语句（CREATE/ALTER/DROP） | `sql` | — |
| `batch_create_tables` | 批量创建表，支持事务模式 | `tables`（表定义数组） | `atomic`（默认 false，逐条执行） |
| `batch_create_indexes` | 批量创建索引，支持事务模式 | `indexes`（索引定义数组） | `atomic`（默认 false） |
| `batch_drop_tables` | 批量删除表 | `table_names`（表名数组） | `if_exists`、`atomic` |
| `batch_drop_indexes` | 批量删除索引 | `index_names`（索引名数组） | `if_exists`、`atomic` |

> **注意**：达梦 DDL 可能触发隐式提交，`atomic=true` 时语义未必等同"单事务全回滚"。推荐使用 `atomic=false`（默认）逐条执行并检查结果。

##### metadata — 元数据查询工具（8 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `list_databases` | 列出服务器上的所有数据库 | — | — |
| `list_schemas` | 列出当前数据库的所有模式（Schema） | — | — |
| `list_tables` | 列出表，可按模式筛选 | — | `schema` |
| `list_views` | 列出视图，可按模式筛选 | — | `schema` |
| `describe_table` | 获取表结构（列名、类型、长度、是否可空、默认值） | `table_name` | — |
| `list_indexes` | 列出指定表的所有索引 | `table_name` | — |
| `search_indexes` | 索引目录检索，支持按表名/索引名模糊搜索 | — | `owner_scope`（USER/ALL）、`schema`、`table_name`、`index_name`、`index_match`（exact/prefix/like） |
| `describe_index` | 获取索引详情（类型、唯一性、状态）及索引列信息 | `index_name` | `table_name`、`owner_scope`、`schema` |

##### advanced — 高级功能工具（4 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `execute_transaction` | 在事务中执行多条 SQL，全成功提交/任意失败回滚 | `statements`（SQL 语句数组） | — |
| `call_procedure` | 调用存储过程 | `procedure_name` | `params`（参数数组） |
| `call_function` | 调用函数并返回结果 | `function_name` | `params`（参数数组） |
| `explain_plan` | 分析 SQL 执行计划，用于性能调优 | `sql` | — |

##### admin — 管理维护工具（3 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `list_users` | 列出所有数据库用户（用户名、状态、创建时间、默认表空间） | — | — |
| `table_statistics` | 获取表统计信息（实际行数、估算行数、块数、最后分析时间） | `table_name` | — |
| `database_info` | 获取数据库服务器基本信息（版本、实例、数据库详情） | — | — |

##### import — 数据导入工具（1 个）

| 工具 | 说明 | 必填参数 | 可选参数 |
|------|------|----------|----------|
| `batch_import_csv` | 使用 DMFLDR 批量并行导入 CSV 文件，支持字段类型映射、空值处理、多种装载模式 | `files`（CSV 文件配置数组） | `dmfldr_path`、`work_dir`、`delimiter`（默认 `,`）、`enclosed_by`、`character_code`（默认 UTF-8）、`rows`（默认 50000）、`direct`（默认 true）、`index_option`（默认 2）、`mode`（APPEND/REPLACE/INSERT）、`max_parallel`（默认 2）、`timeout_seconds`、`errors` |

## 使用方法

### 1. 列出可用工具

```json
{
  "tool": "dm_list_tools",
  "arguments": {
    "category": "query"  // 可选，筛选特定类别
  }
}
```

### 2. 执行操作工具

```json
{
  "tool": "dm_execute",
  "arguments": {
    "tool_name": "query",
    "params": {
      "sql": "SELECT * FROM users LIMIT 10"
    }
  }
}
```

## 环境要求

- Go 1.21+
- 达梦数据库 8.x
- 达梦 Go 驱动（`gitee.com/chunanyong/dm`）

## 安装

```bash
cd dm-mcp
go mod tidy
go build -o dm-mcp.exe
```

## 批量 DDL 说明

达梦 DM 中 **DDL 可能触发隐式提交**（官方手册：一致性 / DM 事务相关语句）。批量建表/建索引/删表/删索引工具提供 `atomic` 参数（默认 `false`，逐条执行并返回 `results`）；`atomic=true` 时用事务包装多条语句，**语义未必等同“单事务全回滚”**。原始 DDL 数组也可用 `batch_execute_ddl` 或 `execute_transaction`。

## CSV 批量导入

`batch_import_csv` 使用达梦 `dmfldr` 快速装载工具导入 CSV，适合大文件或多表并行导入。需要确保运行 MCP 服务的机器可以执行 `dmfldr`，并且 CSV 文件路径对该机器可读。

### 基本用法

```json
{
  "tool": "dm_execute",
  "arguments": {
    "tool_name": "batch_import_csv",
    "params": {
      "dmfldr_path": "/dm/dmdbms/bin/dmfldr",
      "work_dir": "/tmp/dm-csv-import",
      "max_parallel": 2,
      "delimiter": ",",
      "enclosed_by": "\"",
      "character_code": "UTF-8",
      "rows": 50000,
      "direct": true,
      "index_option": 2,
      "mode": "APPEND",
      "files": [
        {
          "csv_file": "/data/users.csv",
          "schema": "APP",
          "table": "USERS",
          "columns": ["ID", "NAME", "CREATED_AT"],
          "skip": 1
        }
      ]
    }
  }
}
```

### 高级用法（带字段类型和空值处理）

```json
{
  "tool": "dm_execute",
  "arguments": {
    "tool_name": "batch_import_csv",
    "params": {
      "dmfldr_path": "/dm/dmdbms/bin/dmfldr",
      "work_dir": "/tmp/dm-csv-import",
      "max_parallel": 2,
      "delimiter": ",",
      "enclosed_by": "\"",
      "character_code": "UTF-8",
      "rows": 50000,
      "direct": true,
      "index_option": 2,
      "mode": "APPEND",
      "errors": 100,
      "files": [
        {
          "csv_file": "/data/company.csv",
          "schema": "SYSDBA",
          "table": "BAS_COMPANY",
          "columns": ["ID", "COMPANYNAME", "STATUS", "CREATEDTIME", "ISDELETED"],
          "column_types": ["CHAR", "CHAR", "INTEGER", "TIMESTAMP 'YYYY-MM-DD HH:MI:SS.FF'", "INTEGER"],
          "null_if": "",
          "skip": 1
        }
      ]
    }
  }
}
```

### 参数说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| files | array | 是 | - | CSV 文件配置数组 |
| dmfldr_path | string | 否 | dmfldr | dmfldr 工具路径 |
| work_dir | string | 否 | 系统临时目录 | 工作目录 |
| delimiter | string | 否 | , | 字段分隔符 |
| enclosed_by | string | 否 | - | 字段引用符 |
| character_code | string | 否 | UTF-8 | 字符编码 |
| rows | int | 否 | 50000 | 每批提交行数 |
| direct | bool | 否 | true | 是否使用直接路径装载 |
| index_option | int | 否 | 2 | 索引选项 (1=维护, 2=跳过, 3=不维护) |
| mode | string | 否 | APPEND | 装载模式 (APPEND/REPLACE/INSERT) |
| max_parallel | int | 否 | 2 | 最大并行数 |
| timeout_seconds | int | 否 | 0 | 超时秒数 (0=不限制) |
| errors | int | 否 | 0 | 最大错误行数 (0=不限制) |

### files 数组参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| csv_file | string | 是 | CSV 文件路径 |
| table | string | 是 | 目标表名 (可带 schema，如 SCHEMA.TABLE) |
| schema | string | 否 | 目标 schema |
| columns | array | 否 | 列名数组 |
| column_types | array | 否 | 字段类型数组 (CHAR, INTEGER, FLOAT, DATE, TIMESTAMP 等) |
| null_if | string | 否 | 空值条件 (空字符串表示将空值转为 NULL) |
| skip | int | 否 | 跳过的行数 (通常为 1，跳过表头) |
| bad_file | string | 否 | 错误记录文件路径 |
| log_file | string | 否 | 日志文件路径 |

### 支持的字段类型

- `CHAR` - 字符串
- `VARCHAR` - 可变长字符串
- `INTEGER` / `INT` - 整数
- `FLOAT` / `DOUBLE` - 浮点数
- `DECIMAL` - 精确数值
- `DATE` - 日期 (需配合 FORMAT)
- `TIMESTAMP` - 时间戳 (需配合 FORMAT)

### 日期格式示例

- `DATE 'YYYY-MM-DD'` - 日期
- `TIMESTAMP 'YYYY-MM-DD HH:MI:SS'` - 时间戳
- `TIMESTAMP 'YYYY-MM-DD HH:MI:SS.FF'` - 带微秒的时间戳

也可以通过环境变量 `DM_FLDR_PATH` 配置 `dmfldr` 路径；当请求参数中提供 `dmfldr_path` 时，以请求参数为准。工具会为每个 CSV 生成独立的 `.ctl`、`.log`、`.bad` 文件，并返回每个文件的执行状态。

## 配置

通过环境变量配置数据库连接：

```bash
export DM_HOST=localhost      # 数据库主机
export DM_PORT=5236           # 数据库端口
export DM_USER=SYSDBA         # 用户名
export DM_PASSWORD=your_pass  # 密码
export DM_DATABASE=DMDB       # 数据库名
export DM_FLDR_PATH=/dm/dmdbms/bin/dmfldr  # 可选，dmfldr 路径
```

## 与 Claude Desktop 集成

在 `claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "dm-database": {
      "command": "path/to/dm-mcp.exe",
      "args": [],
      "env": {
        "DM_HOST": "localhost",
        "DM_PORT": "5236",
        "DM_USER": "SYSDBA",
        "DM_PASSWORD": "your_password",
        "DM_DATABASE": "DMDB"
      }
    }
  }
}
```

## 项目结构

```
dm-mcp/
├── main.go                 # MCP 服务器入口（注册控制工具）
├── handlers.go             # 控制工具处理器
├── config/
│   └── config.go           # 数据库配置
├── database/
│   └── connection.go       # 连接池管理
└── tools/
    ├── registry.go         # 工具注册中心
    ├── query.go            # 查询工具
    ├── execute.go          # DML 工具
    ├── ddl.go              # DDL 工具
    ├── import_csv.go       # CSV 批量导入工具
    ├── metadata.go         # 元数据工具
    ├── advanced.go         # 高级功能工具
    ├── admin.go            # 管理维护工具
    └── utils.go            # 辅助函数
```

## 许可证

MIT License
