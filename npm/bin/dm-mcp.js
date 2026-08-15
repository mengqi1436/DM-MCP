#!/usr/bin/env node
'use strict';

// @dm-mcp/server launcher
// 以 stdio 模式启动从 GitHub Release 下载的达梦 MCP Go 二进制。
// 数据库连接参数通过环境变量传入（DM_HOST/DM_PORT/DM_USER/DM_PASSWORD 等）。

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const platform = process.platform;
const asset = platform === 'win32' ? 'dm-mcp.exe' : 'dm-mcp';
const binPath = path.join(__dirname, '..', 'bin', platform, asset);

if (!fs.existsSync(binPath)) {
  console.error(`[dm-mcp] 未找到二进制 ${binPath}。请确认 npm install 时 postinstall 已执行（npm_config_ignore_scripts=true 会跳过下载）。`);
  process.exit(1);
}

const child = spawn(binPath, ['-transport=stdio'], {
  stdio: 'inherit',
  env: process.env,
});

child.on('error', (err) => {
  console.error(`[dm-mcp] 启动失败: ${err.message}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 0);
  }
});
