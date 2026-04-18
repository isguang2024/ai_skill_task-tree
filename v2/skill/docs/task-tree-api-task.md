# HTTP: 项目 / 任务 / 恢复

**基础地址**：`http://127.0.0.1:8880`

## 通用约定

- HTTP 基础列表返回裸数组 `[...]`。
- HTTP 分页列表返回 envelope：`{ "items": [...], "has_more": true, "next_cursor": "..." }`。
- 下一页直接传 `?cursor=<next_cursor>`。

## 项目

```text
GET    /v1/projects
POST   /v1/projects
GET    /v1/projects/{id}
PATCH  /v1/projects/{id}
DELETE /v1/projects/{id}
GET    /v1/projects/{id}/overview
GET    /v1/projects/{id}/tasks
```

`GET /v1/projects/{id}/overview?view_mode=summary_with_stats` 会返回项目信息 + 任务列表。

## 任务

```text
GET    /v1/tasks
POST   /v1/tasks
GET    /v1/tasks/{id}
PATCH  /v1/tasks/{id}
DELETE /v1/tasks/{id}
DELETE /v1/tasks/{id}/hard
POST   /v1/tasks/{id}/restore
POST   /v1/tasks/{id}/transition
```

- `POST /v1/tasks/{id}/transition` 传 `action`，不是 `status`。
- `GET /v1/tasks/{id}` 默认不带树；要看树请用 `/nodes` 或 `tree-view`。
- `GET /v1/tasks/{id}` 的任务详情里不要默认期待完整上下文。

## 恢复 / 导航 / 收尾

```text
GET    /v1/tasks/{id}/resume
GET    /v1/tasks/{id}/remaining
GET    /v1/tasks/{id}/next-node
GET    /v1/tasks/{id}/context
PATCH  /v1/tasks/{id}/context
GET    /v1/tasks/{id}/wrapup
POST   /v1/tasks/{id}/wrapup
GET    /v1/tasks/{id}/events/stream
GET    /v1/tasks/{id}/memory
PATCH  /v1/tasks/{id}/memory
POST   /v1/tasks/{id}/memory/snapshot
```

## `/resume`

- 只用于恢复工作现场，不是默认查询入口。
- 已知 `node_id` 时，优先读节点详情或节点上下文。
- 只想知道下一步时，优先调 `/next-node`。
- 同一轮里对同一 `task_id` 默认最多一次 `/resume`，除非发生明显状态变化。
- `include` 可选：`events,runs,artifacts,next_node_context,task_memory,stage_memory`。
