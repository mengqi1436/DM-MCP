# 达梦数据库 MCP 工具说明

共 **82 个**工具，分 11 类。单连接模式，无需指定 connection 参数。

## 控制工具 (control) — 2 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `dm_list_tools` | 列出所有可用工具 | `category`（可选） |
| `dm_execute` | 执行指定工具 | `tool_name`（必填）, `params`（可选） |

## 查询工具 (query) — 5 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `query` | 执行 SQL 查询 | `sql`, `params`（可选） |
| `query_one` | 查询单条记录 | `sql`, `params`（可选） |
| `query_paginated` | 分页查询 | `sql`, `page`（默认1）, `page_size`（默认20） |
| `count` | 统计记录数 | `table`（必填）, `where`（可选） |
| `batch_query` | 批量执行多条 SELECT | `queries`（必填）-SQL数组 |

## 数据操作工具 (dml) — 7 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `insert` | 插入一条数据 | `table`（必填）, `data`（必填） |
| `insert_batch` | 批量插入 | `table`（必填）, `rows`（必填） |
| `update` | 更新数据 | `table`（必填）, `data`（必填）, `where`（必填） |
| `update_batch` | 批量更新（多条不同WHERE） | `table`（必填）, `updates`（必填）-数组[{data,where}] |
| `delete` | 删除数据 | `table`（必填）, `where`（必填） |
| `delete_batch` | 批量删除（多条不同WHERE） | `table`（必填）, `wheres`（必填）-条件数组 |
| `merge` | MERGE INTO（UPSERT） | `table`（必填）, `data`（必填）, `match_columns`（必填） |

## 数据定义工具 (ddl) — 14 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `create_table` | 创建表 | `table_name`, `columns` |
| `alter_table` | 修改表结构 | `table_name`, `operation`(ADD/MODIFY/DROP), `column`, `type` |
| `drop_table` | 删除表 | `table_name`, `if_exists`（可选） |
| `create_index` | 创建索引 | `index_name`, `table_name`, `columns`, `unique`（可选） |
| `drop_index` | 删除索引 | `index_name`, `schema`（可选）, `if_exists`（可选） |
| `execute_ddl` | 执行 DDL 语句 | `sql` |
| `batch_create_tables` | 批量创建表 | `tables`, `atomic`（可选） |
| `batch_create_indexes` | 批量创建索引 | `indexes`, `atomic`（可选） |
| `batch_drop_tables` | 批量删除表 | `table_names`, `if_exists`, `atomic` |
| `batch_drop_indexes` | 批量删除索引 | `index_names`, `if_exists`, `atomic` |
| `create_view` | 创建视图 | `view_name`, `sql`, `or_replace`（可选） |
| `drop_view` | 删除视图 | `view_name`, `if_exists`（可选） |
| `create_sequence` | 创建序列 | `seq_name`, `start_with`, `increment_by`, `max_value`, `cache_size` |
| `drop_sequence` | 删除序列 | `seq_name`, `if_exists`（可选） |

> **注意**: DDL 可能触发隐式提交，批量 DDL 推荐 `atomic=false`。

## 元数据工具 (metadata) — 18 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `list_databases` | 列出所有数据库 | 无 |
| `list_schemas` | 列出所有模式 | 无 |
| `list_tables` | 列出表 | `schema`（可选） |
| `list_views` | 列出视图 | `schema`（可选） |
| `list_sequences` | 列出序列 | `schema`（可选） |
| `list_synonyms` | 列出同义词 | `schema`（可选） |
| `list_procedures` | 列出存储过程 | `schema`（可选） |
| `list_functions` | 列出函数 | `schema`（可选） |
| `list_packages` | 列出包 | `schema`（可选） |
| `list_triggers` | 列出触发器 | `schema`（可选）, `table_name`（可选） |
| `describe_table` | 获取表结构 | `table_name`（必填） |
| `batch_describe_tables` | 批量获取多表结构 | `table_names`（必填）, `schema`（可选） |
| `list_indexes` | 列出表的索引 | `table_name`（必填） |
| `search_indexes` | 索引检索 | `owner_scope`, `schema`, `table_name`, `index_name`, `index_match` |
| `describe_index` | 索引详情 | `index_name`（必填）, `table_name`, `owner_scope`, `schema` |
| `list_constraints` | 列出表约束 | `table_name`（必填）, `schema`（可选） |
| `list_table_partitions` | 列出表分区 | `table_name`（必填）, `schema`（可选） |
| `get_table_ddl` | 导出建表DDL（含索引） | `table_name`（必填）, `schema`（可选） |

## 管理工具 (admin) — 12 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `list_users` | 列出所有用户 | 无 |
| `create_user` | 创建用户 | `username`, `password`, `default_tablespace`（可选） |
| `drop_user` | 删除用户 | `username`, `cascade`（可选） |
| `grant_privilege` | 授予权限 | `privilege`, `grantee` |
| `revoke_privilege` | 撤销权限 | `privilege`, `grantee` |
| `create_role` | 创建角色 | `role_name` |
| `drop_role` | 删除角色 | `role_name` |
| `list_roles` | 列出所有角色 | 无 |
| `table_statistics` | 表统计信息 | `table_name`（必填） |
| `database_info` | 数据库基本信息 | 无 |
| `list_tablespaces` | 列出表空间 | 无 |
| `create_tablespace` | 创建表空间 | `tablespace_name`, `datafile`, `size`（可选,默认128MB） |

## 高级工具 (advanced) — 6 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `execute_transaction` | 事务执行多条 SQL | `statements`（必填） |
| `call_procedure` | 调用存储过程 | `procedure_name`（必填）, `params`（可选） |
| `call_function` | 调用函数 | `function_name`（必填）, `params`（可选） |
| `explain_plan` | 分析执行计划 | `sql`（必填） |
| `execute_sql` | 执行任意SQL（自动判断类型） | `sql`（必填） |
| `batch_execute_sql` | 批量执行任意SQL | `statements`（必填）, `atomic`（可选） |

## 监控诊断工具 (monitoring) — 6 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `active_sessions` | 当前活跃会话 | `limit`（可选,默认50） |
| `lock_info` | 锁信息（阻塞关系） | 无 |
| `slow_queries` | 慢查询统计 | `limit`（可选,默认20） |
| `tablespace_usage` | 表空间使用率 | 无 |
| `instance_parameters` | 实例参数（dm.ini） | `name`（可选）-模糊搜索 |
| `session_memory` | 会话内存使用 | 无 |

## 备份恢复工具 (backup) — 4 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `logical_export` | 逻辑导出（dexp） | `output_file`（必填）, `directory`, `owner`, `tables`, `full`, `dexp_path`, `timeout_seconds` |
| `logical_import` | 逻辑导入（dimp） | `input_file`（必填）, `directory`, `owner`, `tables`, `full`, `dimp_path`, `timeout_seconds` |
| `physical_backup` | 物理备份（dmrman） | `backup_dir`（必填）, `dm_ini_path`（必填）, `backup_name`, `dmrman_path`, `timeout_seconds` |
| `physical_restore` | 物理恢复（dmrman） | `backup_dir`（必填）, `dm_ini_path`（必填）, `confirm=true`, `dmrman_path`, `timeout_seconds` |

## 数据导入导出工具 (import) — 2 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `batch_import_csv` | DMFLDR 批量并行导入 CSV | `files`（必填）+ 多个可选参数 |
| `export_table_data` | 导出表数据为INSERT/JSON | `table_name`（必填）, `format`（json/insert）, `where`, `limit` |

## 实例管理工具 (instance) — 6 个

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `create_database` | dminit 初始化实例 | `path`, `sysdba_pwd`, `sysauditor_pwd` + 可选参数 |
| `delete_database` | 删除实例目录 | `database_dir`, `confirm=true` + 可选参数 |
| `start_database_service` | 启动服务 | `instance_name` 或 `service_name` |
| `stop_database_service` | 停止服务 | `instance_name` 或 `service_name` |
| `restart_database_service` | 重启服务 | `instance_name` 或 `service_name` |
| `database_service_status` | 服务状态 | `instance_name` 或 `service_name` |
