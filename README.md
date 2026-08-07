# 达梦数据库 MCP 工具

一个基于 Go 语言和 MCP（Model Context Protocol）开发的达梦数据库操作工具服务器。

## 架构设计

**单实例单连接**：每个 MCP 进程只连接一个达梦数据库。需要操作多个数据库时，由 mcphub 启动多个 MCP 实例，每个实例配置不同的环境变量。

## 功能特性

共 **82 个工具**，分 11 类。详见 [TOOLS.md](TOOLS.md)。

| 类别 | 数量 | 说明 |
|------|------|------|
| control | 2 | 工具发现和统一执行入口 |
| query | 5 | SELECT 查询、分页、统计、批量查询 |
| dml | 7 | 增删改、批量操作、MERGE/UPSERT |
| ddl | 14 | 表/索引/视图/序列的创建删除、批量DDL |
| metadata | 18 | 库/模式/表/视图/序列/同义词/存储过程/函数/包/触发器/约束/分区/DDL导出 |
| admin | 12 | 用户/角色/权限/表空间管理、统计信息 |
| advanced | 6 | 事务、存储过程调用、执行计划、通用SQL |
| monitoring | 6 | 活跃会话、锁、慢查询、表空间使用率、实例参数、内存 |
| backup | 4 | 逻辑导出导入(dexp/dimp)、物理备份恢复(dmrman) |
| import | 2 | DMFLDR批量CSV导入、表数据导出 |
| instance | 6 | dminit初始化、服务启停、实例删除 |

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

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `DM_HOST` | 否 | `localhost` | 数据库主机地址 |
| `DM_PORT` | 否 | `5236` | 数据库端口 |
| `DM_USER` | 否 | `SYSDBA` | 用户名 |
| `DM_PASSWORD` | 是 | — | 密码 |
| `DM_DATABASE` | 否 | `DMDB` | 数据库名 |
| `DM_SCHEMA` | 否 | — | 默认 Schema |
| `DM_FLDR_PATH` | 否 | `dmfldr` | dmfldr 工具路径（CSV 导入用） |
| `DM_EXP_PATH` | 否 | `dexp` | dexp 工具路径（逻辑导出用） |
| `DM_IMP_PATH` | 否 | `dimp` | dimp 工具路径（逻辑导入用） |
| `DM_RMAN_PATH` | 否 | `dmrman` | dmrman 工具路径（物理备份用） |

## MCP 配置

### 单数据库

```json
{
  "mcpServers": {
    "dm-database": {
      "command": "D:/path/to/dm-mcp.exe",
      "env": {
        "DM_HOST": "localhost",
        "DM_PORT": "5236",
        "DM_USER": "SYSDBA",
        "DM_PASSWORD": "your_password",
        "DM_DATABASE": "DMDB",
        "DM_SCHEMA": "APP"
      }
    }
  }
}
```

### 多数据库（mcphub 多实例）

```json
{
  "mcpServers": {
    "dm-prod": {
      "command": "D:/path/to/dm-mcp.exe",
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
      "command": "D:/path/to/dm-mcp.exe",
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

## 项目结构

```
dm-mcp/
├── main.go                 # MCP 服务器入口
├── handlers.go             # 控制工具处理器
├── operation_tools.go      # 操作工具注册到 MCP
├── config/
│   └── config.go           # 数据库配置（DM_* 环境变量）
├── database/
│   └── connection.go       # 单例连接池
└── tools/
    ├── registry.go         # 工具注册中心
    ├── utils.go            # 辅助函数
    ├── query.go            # 查询工具
    ├── execute.go          # DML 工具
    ├── ddl.go              # DDL 工具
    ├── schema_objects.go   # 视图/序列/同义词/存储过程/函数/包/触发器/约束/分区
    ├── metadata.go         # 元数据工具
    ├── advanced.go         # 高级功能工具
    ├── batch_ops.go        # 批量操作和通用SQL
    ├── admin.go            # 管理维护工具
    ├── admin_extended.go   # 用户/角色/权限/表空间管理
    ├── monitoring.go       # 监控诊断工具
    ├── backup.go           # 备份恢复工具
    ├── instance.go         # 实例与服务工具
    └── import_csv.go       # CSV 批量导入工具
```

## 许可证

MIT License
