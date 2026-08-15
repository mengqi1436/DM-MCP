# @dm-mcp/server

达梦数据库（DM8）MCP 服务器（stdio 传输）的 npm 包装包。

安装时（postinstall）自动从 [GitHub Releases](https://github.com/mengqi1436/DM-MCP/releases) 下载对应平台（win32/linux, x64）的 Go 二进制。

## 安装

```bash
npm install @dm-mcp/server
```

## 配置（环境变量）

| 变量 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `DM_PASSWORD` | 是 | — | 数据库密码 |
| `DM_HOST` | 否 | `localhost` | 主机 |
| `DM_PORT` | 否 | `5236` | 端口 |
| `DM_USER` | 否 | `SYSDBA` | 用户 |
| `DM_SCHEMA` | 否 | — | 默认 Schema |
| `DM_MAX_OPEN_CONNS` | 否 | `16` | 连接池上限 |
| `DM_QUERY_TIMEOUT` | 否 | `30s` | 单请求超时 |
| `DM_QUERY_LIMIT` | 否 | `1000` | 查询行数上限 |

## 使用

### 作为 MCP 服务器（stdio）

```bash
DM_PASSWORD=your_password npx dm-mcp
```

在 Claude Desktop / Claude Code / Cursor 的 MCP 配置中：

```json
{
  "mcpServers": {
    "dm-database": {
      "command": "npx",
      "args": ["-y", "@dm-mcp/server"],
      "env": {
        "DM_HOST": "localhost",
        "DM_PORT": "5236",
        "DM_USER": "SYSDBA",
        "DM_PASSWORD": "your_password"
      }
    }
  }
}
```

### 直接调用二进制

```bash
node_modules/.bin/dm-mcp
```

## 支持平台

- `linux` / `x64`
- `win32` / `x64`

## 许可

MIT
