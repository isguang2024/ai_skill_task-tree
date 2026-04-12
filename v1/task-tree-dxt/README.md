# Task Tree V1 (Core) DXT Extension

Claude Desktop 扩展，通过 stdio-to-HTTP 代理连接本地 Task Tree V1 MCP 服务。

## 前提条件

- Task Tree V1 服务已在本地运行（`http://127.0.0.1:8879`）
- Node.js >= 16.0.0

## 安装

1. 确保 Task Tree V1 对应的 `task-tree-service.exe serve` 正在运行
2. 将 `task-tree.dxt` 拖拽到 Claude Desktop 的 Extensions 安装区域
3. 重启 Claude Desktop 会话

## 工作原理

Claude Desktop 仅支持通过 stdio 与 MCP 服务器通信，而 Task Tree V1 当前暴露的是 HTTP MCP 服务器。

`proxy.mjs` 作为 stdio-to-HTTP 桥接：
- 从 stdin 读取 JSON-RPC 请求
- 转发到 `http://127.0.0.1:8879/mcp`
- 将 HTTP 响应写回 stdout

## 文件说明

| 文件 | 说明 |
|------|------|
| `manifest.json` | DXT 扩展清单，定义扩展元数据和启动方式 |
| `proxy.mjs` | stdio-to-HTTP MCP 代理脚本 |
| `task-tree.dxt` | 打包后的扩展文件（zip 格式） |

## 目录约定

- `task-tree/v1/task-tree-dxt/` 对应统一仓库中的 V1 版本
- `task-tree/v2/task-tree-dxt/` 对应统一仓库中的 V2 版本

两个版本各有独立的 DXT 目录和代理配置，防止冲突。

## 重新打包

修改 `manifest.json` 或 `proxy.mjs` 后，重新打包：

```powershell
cd task-tree-dxt
Compress-Archive -Path 'manifest.json','proxy.mjs' -DestinationPath 'task-tree.zip' -Force
mv task-tree.zip task-tree.dxt
```

## 故障排查

**服务未启动**：确认 `curl http://127.0.0.1:8879/healthz` 返回 `{"ok":true}`

**代理测试**：
```bash
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}\n' | node proxy.mjs
```

**查看 MCP 日志**：`%APPDATA%\Claude\logs\mcp.log`
