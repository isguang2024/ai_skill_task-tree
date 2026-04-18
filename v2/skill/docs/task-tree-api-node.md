# HTTP: 节点 / Run / 产物

**基础地址**：`http://127.0.0.1:8880`

## 节点

```text
GET    /v1/tasks/{id}/nodes
POST   /v1/tasks/{id}/nodes
POST   /v1/tasks/{id}/nodes/batch
GET    /v1/tasks/{id}/tree-view
GET    /v1/nodes/{id}
PATCH  /v1/nodes/{id}
POST   /v1/nodes/reorder
POST   /v1/nodes/{id}/move
POST   /v1/nodes/{id}/retype
```

## 执行

```text
POST   /v1/nodes/{id}/claim-and-start-run
POST   /v1/nodes/{id}/claim
POST   /v1/nodes/{id}/release
POST   /v1/nodes/{id}/progress
POST   /v1/nodes/{id}/complete
POST   /v1/nodes/{id}/block
POST   /v1/nodes/{id}/transition
```

- `block` 推荐改用 `transition`。
- `complete` 的 `memory` 里只写结构化字段，例如 `summary_text`、`decisions`、`evidence`。

## 上下文

```text
GET    /v1/nodes/{id}/context
GET    /v1/tasks/{id}/nodes/{nid}/resume-context
GET    /v1/nodes/{id}/memory
PATCH  /v1/nodes/{id}/memory
POST   /v1/nodes/{id}/memory/snapshot
```

## Run

```text
POST   /v1/nodes/{id}/runs
GET    /v1/nodes/{id}/runs
GET    /v1/runs/{id}
POST   /v1/runs/{id}/finish
POST   /v1/runs/{id}/logs
```

- `GET /v1/runs/{id}` 默认不带日志；需要时再显式请求。
- `result` 合法值：`done`、`canceled`。

## 产物

```text
GET    /v1/tasks/{id}/artifacts
GET    /v1/nodes/{id}/artifacts
POST   /v1/tasks/{id}/artifacts
POST   /v1/tasks/{id}/artifacts/upload
GET    /v1/artifacts/{id}/download
```
