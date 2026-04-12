# Task Tree DXT Extension

Claude Desktop 扩展，通过 stdio-to-HTTP 代理连接本地 `task-tree-v2` MCP 服务。

## 前提条件

- `task-tree-v2` 服务已在本地运行（`http://127.0.0.1:8880`）
- Node.js >= 16.0.0

## 安装

1. 确保 `task-tree-v2` 对应的 `task-tree-service.exe serve` 正在运行
2. 将 `task-tree.dxt` 拖拽到 Claude Desktop 的 Extensions 安装区域
3. 重启 Claude Desktop 会话

## 工作原理

Claude Desktop 仅支持通过 stdio 与 MCP 服务器通信，而 `task-tree-v2` 当前暴露的是 HTTP MCP 服务器。

`proxy.mjs` 作为 stdio-to-HTTP 桥接：
- 从 stdin 读取 JSON-RPC 请求
- 转发到 `http://127.0.0.1:8880/mcp`
- 将 HTTP 响应写回 stdout

## 文件说明

| 文件 | 说明 |
|------|------|
| `manifest.json` | V2 DXT 扩展清单 |
| `proxy.mjs` | stdio-to-HTTP MCP 代理脚本 |
| `task-tree.dxt` | 打包后的 V2 扩展文件 |

## 重新打包

修改 `manifest.json` 或 `proxy.mjs` 后，重新打包：

```powershell
cd task-tree-v2/task-tree-dxt
Compress-Archive -Path 'manifest.json','proxy.mjs','README.md' -DestinationPath 'task-tree.zip' -Force
mv task-tree.zip task-tree.dxt
```

重新打包前请先确认：

- `proxy.mjs` 指向 `http://127.0.0.1:8880/mcp`

## 故障排查

**服务未启动**：确认 `curl http://127.0.0.1:8880/healthz` 返回 `{"ok":true}`

**代理测试**：
```bash
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}\n' | node proxy.mjs
```

**查看 MCP 日志**：`%APPDATA%\Claude\logs\mcp.log`
