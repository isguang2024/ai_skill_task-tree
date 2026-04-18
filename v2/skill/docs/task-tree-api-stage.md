# HTTP: 阶段 / Stage Memory

**基础地址**：`http://127.0.0.1:8880`

```text
GET    /v1/tasks/{id}/stages
POST   /v1/tasks/{id}/stages
POST   /v1/tasks/{id}/stages/batch
POST   /v1/tasks/{id}/stages/{sid}/activate
```

- 创建阶段字段名是 `title`，不是 `name`。

## Stage Memory

```text
GET    /v1/stages/{id}/memory
PATCH  /v1/stages/{id}/memory
POST   /v1/stages/{id}/memory/snapshot
```

- 这些是 HTTP only，没有对应的 MCP 工具。
