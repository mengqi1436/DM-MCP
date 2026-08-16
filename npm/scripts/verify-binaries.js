// 校验 npm 包内的预编译二进制是否齐全（npm pack/publish 前执行）。
// 二进制由 Release workflow 从构建产物拷贝进来，本地开发时可能缺失——
// 缺失时给出提示而不是阻塞（发布流程才会真正打包它们）。
'use strict';

const fs = require('fs');
const path = require('path');

const EXPECTED = [
  'dm-mcp2-linux-x64',
  'dm-mcp2-linux-arm64',
  'dm-mcp2-darwin-x64',
  'dm-mcp2-darwin-arm64',
  'dm-mcp2-win32-x64.exe',
];

const binDir = path.join(__dirname, '..', 'bin');
const missing = EXPECTED.filter((f) => !fs.existsSync(path.join(binDir, f)));

if (missing.length > 0) {
  console.warn(`[dm-mcp] 警告: bin/ 目录缺少二进制: ${missing.join(', ')}`);
  console.warn('[dm-mcp] 正式发布前请运行 Release workflow 构建并拷贝产物。');
}
