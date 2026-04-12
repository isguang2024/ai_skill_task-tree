# 变更检查清单

- 当前路径是否为 `task-tree/v2/`
- 是否使用 V2 端口 `8880`
- HTTP 改动是否同步检查 MCP
- 后端改动后是否执行 `go test ./...`
- 前端改动后是否执行 `npm run build`
- 若涉及 DXT，`proxy.mjs` 是否仍指向 `http://127.0.0.1:8880/mcp`
