# HTTP: 全局 / 搜索 / 管理

**基础地址**：`http://127.0.0.1:8880`

```text
GET    /healthz
GET    /v1/work-items
GET    /v1/search
GET    /v1/smart-search
GET    /v1/events
POST   /v1/import-plan
POST   /v1/admin/sweep-leases
POST   /v1/admin/empty-trash
POST   /v1/admin/rebuild-index
```

- `GET /v1/search` 是旧入口，内部转发到 `smart-search`。
- `POST /v1/import-plan` 的 `data` 必须是字符串。
- `smart-search` 支持 `q`、`scope=task/node/memory/all`、`task_id`、`limit`。
