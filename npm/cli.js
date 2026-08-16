#!/usr/bin/env node
// dm-mcp — 达梦数据库 MCP 服务器 npm 启动器
// 从包内 bin/ 目录选择当前平台/架构的二进制，并以 stdio 模式启动（MCP 默认传输）。
'use strict';

const { spawn } = require('child_process');
const path = require('path');

function resolveBinary() {
  const platform = process.platform; // win32 | darwin | linux
  const arch = process.arch; // x64 | arm64

  let name;
  if (platform === 'win32') {
    if (arch === 'x64') name = 'dm-mcp2-win32-x64.exe';
    else throw new Error(`不支持的 Windows 架构: ${arch}（仅支持 x64）`);
  } else if (platform === 'linux') {
    if (arch === 'x64') name = 'dm-mcp2-linux-x64';
    else if (arch === 'arm64') name = 'dm-mcp2-linux-arm64';
    else throw new Error(`不支持的 Linux 架构: ${arch}（仅支持 x64/arm64）`);
  } else if (platform === 'darwin') {
    if (arch === 'x64') name = 'dm-mcp2-darwin-x64';
    else if (arch === 'arm64') name = 'dm-mcp2-darwin-arm64';
    else throw new Error(`不支持的 macOS 架构: ${arch}（仅支持 x64/arm64）`);
  } else {
    throw new Error(`不支持的平台: ${platform}（仅支持 win32/linux/darwin）`);
  }

  return path.join(__dirname, 'bin', name);
}

const bin = resolveBinary();
const child = spawn(bin, ['-transport=stdio'], { stdio: 'inherit' });

child.on('error', (err) => {
  if (err.code === 'ENOENT') {
    console.error(
      `未找到二进制 ${bin}\n` +
      '请确认已安装完整版本的 dm-mcp 包（npm install dm-mcp），' +
      '或使用 GitHub Releases 中的预编译二进制。'
    );
  } else {
    console.error(`启动 dm-mcp 失败: ${err.message}`);
  }
  process.exit(1);
});

child.on('exit', (code, signal) => {
  process.exit(code ?? 1);
});
