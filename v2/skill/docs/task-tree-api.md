# Task Tree V2 — 完整 HTTP API 参考

> 本文件是按需查阅的参考文档，不在每次对话中加载。需要时用 `Read` 工具读取。

## 项目

```
GET    /v1/projects                       — 列出项目
POST   /v1/projects                       — 创建项目
GET    /v1/projects/{id}                  — 项目详情
PATCH  /v1/projects/{id}                  — 更新项目
DELETE /v1/projects/{id}                  — 删除项目
GET    /v1/projects/{id}/overview         — 项目概览
GET    /v1/projects/{id}/tasks            — 项目下的任务列表
```

## 任务

```
GET    /v1/tasks                          — 列出任务
POST   /v1/tasks                          — 创建任务（支持 stages + nodes 一步到位）
GET    /v1/tasks/{id}                     — 任务详情
PATCH  /v1/tasks/{id}                     — 更新任务
DELETE /v1/tasks/{id}                     — 软删除
DELETE /v1/tasks/{id}/hard                — 硬删除
POST   /v1/tasks/{id}/restore             — 恢复
POST   /v1/tasks/{id}/transition          — 状态流转
GET    /v1/tasks/{id}/resume              — 任务上下文（核心聚合查询）
GET    /v1/tasks/{id}/remaining           — 剩余进度
GET    /v1/tasks/{id}/next-node           — 下一个可执行节点
GET    /v1/tasks/{id}/context             — 任务上下文快照（架构决策/参考文件）
PATCH  /v1/tasks/{id}/context             — 更新任务上下文快照
GET    /v1/tasks/{id}/wrapup             — 获取任务收尾总结
POST   /v1/tasks/{id}/wrapup             — 写入/更新任务收尾总结
GET    /v1/tasks/{id}/events/stream       — SSE 事件流
```

## 任务 Memory

```
GET    /v1/tasks/{id}/memory              — 获取（与 /context 返回相同数据）
PATCH  /v1/tasks/{id}/memory              — 更新
POST   /v1/tasks/{id}/memory/snapshot     — 快照
```

## 阶段

```
GET    /v1/tasks/{id}/stages              — 列出阶段
POST   /v1/tasks/{id}/stages              — 创建阶段
POST   /v1/tasks/{id}/stages/batch        — 批量创建阶段
POST   /v1/tasks/{id}/stages/{sid}/activate — 激活阶段
GET    /v1/stages/{id}/memory             — 阶段 Memory
PATCH  /v1/stages/{id}/memory             — 更新阶段 Memory
POST   /v1/stages/{id}/memory/snapshot    — 快照阶段 Memory
```

## 节点

```
GET    /v1/tasks/{id}/nodes               — 列出节点，始终返回 {items:[...], has_more:...}
POST   /v1/tasks/{id}/nodes               — 创建节点
POST   /v1/tasks/{id}/nodes/batch         — 批量创建节点
GET    /v1/tasks/{id}/tree-view           — 树形可视化（文本树）
GET    /v1/nodes/{id}                     — 节点详情
PATCH  /v1/nodes/{id}                     — 更新节点
GET    /v1/nodes/{id}/context             — 节点上下文（核心）
GET    /v1/tasks/{id}/nodes/{nid}/resume-context — 节点级恢复上下文（祖先链 + Memory + 最近运行）
POST   /v1/nodes/{id}/transition          — 状态流转（block/pause/reopen/cancel/unblock）
POST   /v1/nodes/{id}/move                — 移动节点
POST   /v1/nodes/reorder                  — 重排序
POST   /v1/nodes/{id}/progress            — 上报进度
POST   /v1/nodes/{id}/complete            — 完成节点（支持内联 memory + 自动 next_node）
POST   /v1/nodes/{id}/block               — 标记阻塞
POST   /v1/nodes/{id}/claim-and-start-run — 领取+开始运行
POST   /v1/nodes/{id}/claim               — 领取
POST   /v1/nodes/{id}/release             — 释放
POST   /v1/nodes/{id}/retype              — 类型转换
```

### 节点列表查询参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `view_mode` | 返回字段详略：`slim`(5字段) / `summary`(13字段) / `detail`(全量) / `events`(summary+最近事件) | `?view_mode=slim` |
| `kind` | 按类型过滤，逗号分隔 | `?kind=leaf` |
| `status` | 按状态过滤，逗号分隔 | `?status=ready,running` |
| `depth` | 只返回指定深度 | `?depth=0` 只返回顶层 |
| `max_depth` | 返回不超过该深度 | `?max_depth=1` |
| `has_children` | 按是否有子节点过滤 | `?has_children=false` 只返回叶子 |
| `limit` | 返回数量上限（默认 100） | `?limit=5` |
| `filter_mode` | 预设过滤：`all` / `focus` / `active` / `blocked` / `done` | `?filter_mode=active` |
| `sort_by` | 排序字段：`path` / `updated_at` / `created_at` / `status` / `progress` | `?sort_by=updated_at` |
| `sort_order` | `asc` / `desc` | `?sort_order=desc` |
| `cursor` | 分页游标 | 从上一次响应的 `next_cursor` 获取 |

## 节点 Memory

```
GET    /v1/nodes/{id}/memory              — 获取
PATCH  /v1/nodes/{id}/memory              — 更新
POST   /v1/nodes/{id}/memory/snapshot     — 快照
```

## 运行

```
POST   /v1/nodes/{id}/runs                — 启动运行
GET    /v1/nodes/{id}/runs                — 列出运行
GET    /v1/runs/{id}                      — 运行详情
POST   /v1/runs/{id}/finish               — 结束运行
POST   /v1/runs/{id}/logs                 — 追加日志
```

## 产物

```
GET    /v1/tasks/{id}/artifacts           — 任务产物
GET    /v1/nodes/{id}/artifacts           — 节点产物
POST   /v1/tasks/{id}/artifacts           — 创建产物
POST   /v1/tasks/{id}/artifacts/upload    — 上传产物
GET    /v1/artifacts/{id}/download        — 下载产物
```

## 全局

```
GET    /healthz                           — 健康检查，返回 {"ok":true}
GET    /v1/work-items                     — 待执行工作项
GET    /v1/search                         — 搜索（LIKE，已废弃，内部转发到 smart-search）
GET    /v1/smart-search                   — 全文检索（FTS5 + BM25）
GET    /v1/events                         — 事件列表
POST   /v1/import-plan                    — 导入 Markdown/YAML/JSON 计划
POST   /v1/admin/sweep-leases             — 清理过期 lease
POST   /v1/admin/empty-trash              — 清空回收站
POST   /v1/admin/rebuild-index            — 重建全文检索索引
```

## AI 对话（需配置 API Key）

```
GET    /ai/status                         — 查看 AI 配置状态（provider/model/wire_api）
POST   /ai/chat                           — AI 对话（同步，返回完整响应）
POST   /ai/chat/stream                    — AI 对话（SSE 流式）
POST   /ai/clear                          — 清除会话历史（body: {"session_id":"..."} 可选）
```

> AI 接口需在 `backend/ai.env.txt` 配置 `OPENAI_API_KEY` 或 `ANTHROPIC_API_KEY`。
> `AI_BASE_URL` 填提供商根地址（不含 `/v1`），代码内部会自动拼接。

## MCP 与 HTTP 返回格式差异

MCP 工具层对返回数据做了裁剪优化，与 HTTP 返回有以下差异：

| 接口 | HTTP 返回 | MCP 返回 |
|------|----------|---------|
| `list_nodes` | 无参数返回全量字段 | 默认 `view_mode=summary`（13 字段） |
| `list_tasks` | 全量字段含 wrapup_summary | 裁剪为 12 字段，去掉 wrapup_summary/goal/metadata |
| `list_stages` | 全量 node 字段（33 字段） | 裁剪为 10 字段 + 包装 `{items:[...]}` |
| `list_events` | 默认 limit=100 | 默认 limit=20 |
| `work_items` | 全量 node 字段 | 裁剪为 10 字段 + 包装 `{items:[...]}` |
| `resume` | 全量 task/memory | task 裁剪为核心字段，task_memory 只保留 summary/decisions/risks/next_actions |
| 所有 MCP 输出 | 含 null 字段 | null 和空 map 字段被自动移除（omitEmpty） |

## 调用注意事项

| 接口 | 注意 |
|------|------|
| `POST /v1/tasks/{id}/stages` | 字段名是 `title`，不是 `name` |
| `POST /v1/import-plan` | `data` 字段必须是 **JSON 字符串**，不是对象。`apply` 为 false 时仅预演不落库 |
| `POST /v1/nodes/{id}/transition` | 必须传 `action` 字段（block/pause/reopen/unblock/cancel），不是 `status` |
| `POST /v1/tasks/{id}/transition` | 必须传 `action` 字段（pause/reopen/cancel） |
| `POST /v1/nodes/{id}/complete` `block` `retype` | 只支持 `kind=leaf` 节点 |
| `POST /v1/nodes/{id}/retype` | 只能将 `kind=group` 转为 `leaf`，不能转 linked_task 或 task |
| `GET /v1/artifacts/{id}/download` | 只有通过 upload 上传的本地产物才可下载；URL 类型产物返回 404 |
| `POST /v1/nodes/reorder` | `node_ids` 不能为空数组 |
| MCP 过滤参数 | `kind`、`status`、`type` 等数组参数支持字符串和数组两种写法：`"leaf"` 或 `["leaf"]` |
