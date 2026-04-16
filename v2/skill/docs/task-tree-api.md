# Task Tree V2 — HTTP API 参考

> 本文件按需查阅，不在每次对话中加载。

**基础地址**：`http://127.0.0.1:8880`

## 通用约定

### 响应格式

所有接口返回 JSON。列表接口统一返回：

```json
{
  "items": [...],
  "has_more": true,
  "next_cursor": "2024-01-15T10:30:00Z|node_abc123"
}
```

### 分页（cursor）

下一页只需传 `?cursor=<next_cursor>` 值。不要自己拼 cursor 格式。

### 错误响应

```json
{
  "error": "run status is finished",
  "code": 409
}
```

常见错误码：

| 状态码 | 含义 | 常见场景 |
|--------|------|---------|
| 400 | 参数错误 | 缺少必填字段、字段格式错误 |
| 404 | 不存在 | task/node/run ID 无效 |
| 409 | 冲突 | 节点已有 active run、run 已结束、状态流转不合法 |
| 500 | 服务器错误 | 内部异常 |

---

## 项目

```
GET    /v1/projects                    # 列出项目
POST   /v1/projects                    # 创建项目
GET    /v1/projects/{id}               # 项目详情
PATCH  /v1/projects/{id}               # 更新项目
DELETE /v1/projects/{id}               # 删除项目
GET    /v1/projects/{id}/overview      # 项目概览（推荐 ?view_mode=summary_with_stats）
GET    /v1/projects/{id}/tasks         # 项目下的任务列表
```

### 示例：项目概览

```
GET /v1/projects/proj_abc/overview?view_mode=summary_with_stats
```

响应包含项目信息 + 任务列表（含统计摘要）。

---

## 任务

```
GET    /v1/tasks                       # 列出任务（?status=&project_id=&q=）
POST   /v1/tasks                       # 创建任务（可含 stages + nodes）
GET    /v1/tasks/{id}                  # 任务详情（默认不带树）
PATCH  /v1/tasks/{id}                  # 更新任务
DELETE /v1/tasks/{id}                  # 软删除（移入回收站）
DELETE /v1/tasks/{id}/hard             # 硬删除（不可恢复）
POST   /v1/tasks/{id}/restore         # 从回收站恢复
POST   /v1/tasks/{id}/transition      # 状态流转（传 action，不是 status）
```

### 示例：创建任务（含骨架）

```
POST /v1/tasks
Content-Type: application/json

{
  "title": "重构用户模块",
  "goal": "将用户模块拆分为独立服务",
  "project_id": "proj_abc",
  "stages": [
    {"title": "数据层", "key": "data"},
    {"title": "API层", "key": "api"}
  ],
  "nodes": [
    {
      "title": "迁移 user 表",
      "kind": "leaf",
      "stage_key": "data",
      "key": "migrate-table",
      "instruction": "将 user 表迁移到新 schema",
      "estimate": "2h"
    },
    {
      "title": "更新 repository",
      "kind": "leaf",
      "stage_key": "data",
      "depends_on_keys": ["migrate-table"]
    }
  ]
}
```

### 示例：任务状态流转

```
POST /v1/tasks/{id}/transition
Content-Type: application/json

{"action": "pause"}
```

合法 action：`pause`、`reopen`、`cancel`

### 恢复 / 导航 / 收尾

```
GET    /v1/tasks/{id}/resume           # 恢复上下文（仅恢复现场）
GET    /v1/tasks/{id}/remaining        # 剩余统计
GET    /v1/tasks/{id}/next-node        # 推荐下一可执行节点
GET    /v1/tasks/{id}/context          # 任务上下文快照
PATCH  /v1/tasks/{id}/context          # 更新任务上下文快照
GET    /v1/tasks/{id}/wrapup           # 读取收尾总结
POST   /v1/tasks/{id}/wrapup           # 写入收尾总结
GET    /v1/tasks/{id}/events/stream    # SSE 事件流
```

### `/resume` 参数详解

**默认返回**（轻量包）：

- `task` — 任务基础信息
- `task_memory_summary` — 任务 Memory 摘要
- `current_stage` — 当前激活阶段
- `tree` — 节点树
- `remaining` — 剩余统计
- `recommended_action` — 推荐下一步
- `next_node_summary` — 下一节点摘要

**调用约束**：

- `/resume` 只用于恢复工作现场，不是默认查询入口。
- 已知 `node_id` 时，优先读取节点详情或节点上下文。
- 只想知道下一步时，优先调 `/next-node`。
- 只想看局部树时，优先调 `/nodes`、`focus_nodes` 对应能力。
- 同一轮里对同一 `task_id` 默认最多一次 `/resume`，除非发生重大状态变化。

**重上下文**（通过 `include` 显式请求，逗号分隔）：

```
GET /v1/tasks/{id}/resume?include=events,runs,next_node_context
```

| include 值 | 追加内容 |
|------------|---------|
| `events` | 近期事件流 |
| `runs` | 近期 Run 列表 |
| `artifacts` | 产物列表 |
| `next_node_context` | 下一节点的完整上下文 |
| `task_memory` | 任务 Memory 完整内容 |
| `stage_memory` | 阶段 Memory 完整内容 |

**树过滤参数**（与 list_nodes 相同）：

| 参数 | 说明 |
|------|------|
| `view_mode` | `slim` / `summary` / `detail` / `events` |
| `filter_mode` | `all` / `focus` / `active` / `blocked` / `done` |
| `status` | 按状态过滤 |
| `kind` | 按节点类型过滤 |
| `q` | 关键词搜索 |
| `depth` / `max_depth` | 深度限制 |
| `parent_node_id` | 某个父节点的直接 children |
| `subtree_root_node_id` | 某个子树 |
| `max_relative_depth` | 相对子树根的深度限制 |
| `has_children` | 按是否有子节点过滤 |
| `limit` / `cursor` | 分页 |
| `sort_by` | `path` / `updated_at` / `created_at` / `status` / `progress` |
| `sort_order` | `asc` / `desc` |

---

## Task Memory（HTTP only）

```
GET    /v1/tasks/{id}/memory           # 读取任务 Memory
PATCH  /v1/tasks/{id}/memory           # 更新任务 Memory
POST   /v1/tasks/{id}/memory/snapshot  # 创建任务 Memory 快照
```

> 注意：这些是 **HTTP only**，没有对应的 MCP 工具。任务上下文快照用 `task_tree_get_task_context` / `task_tree_patch_task_context`。

---

## 阶段

```
GET    /v1/tasks/{id}/stages                    # 列出阶段
POST   /v1/tasks/{id}/stages                    # 创建阶段（字段名是 title，不是 name）
POST   /v1/tasks/{id}/stages/batch              # 批量创建阶段
POST   /v1/tasks/{id}/stages/{sid}/activate     # 激活阶段
```

### Stage Memory（HTTP only）

```
GET    /v1/stages/{id}/memory           # 读取阶段 Memory
PATCH  /v1/stages/{id}/memory           # 更新阶段 Memory
POST   /v1/stages/{id}/memory/snapshot  # 创建阶段 Memory 快照
```

---

## 节点

### CRUD

```
GET    /v1/tasks/{id}/nodes            # 节点列表（支持全部过滤/分页参数）
POST   /v1/tasks/{id}/nodes            # 创建单个节点
POST   /v1/tasks/{id}/nodes/batch      # 批量创建节点（事务原子）
GET    /v1/tasks/{id}/tree-view        # 缩进树文本视图
GET    /v1/nodes/{id}                  # 节点详情
PATCH  /v1/nodes/{id}                  # 更新节点
POST   /v1/nodes/reorder              # 重排同级节点
POST   /v1/nodes/{id}/move            # 移动节点
POST   /v1/nodes/{id}/retype          # group → leaf（仅无子节点时）
```

### 执行

```
POST   /v1/nodes/{id}/claim-and-start-run  # 领取 + 创建 Run（推荐）
POST   /v1/nodes/{id}/claim               # 仅领取 lease
POST   /v1/nodes/{id}/release             # 释放 lease
POST   /v1/nodes/{id}/progress            # 上报进度
POST   /v1/nodes/{id}/complete            # 完成节点
POST   /v1/nodes/{id}/block               # 标记阻塞（推荐用 transition）
POST   /v1/nodes/{id}/transition          # 状态流转
```

### 示例：完成节点

```
POST /v1/nodes/{id}/complete
Content-Type: application/json

{
  "memory": {
    "summary_text": "完成 user 表迁移，新增 3 列，验证通过",
    "execution_log": "1. 创建迁移文件\n2. 执行迁移\n3. 运行测试",
    "evidence": ["migrations/20240115_user.sql", "go test ./... PASS"]
  }
}
```

### 上下文

```
GET    /v1/nodes/{id}/context                      # 节点上下文（preset=summary/memory/work/full）
GET    /v1/tasks/{id}/nodes/{nid}/resume-context   # 节点级 resume 上下文
```

### Node Memory

```
GET    /v1/nodes/{id}/memory           # 读取节点 Memory
PATCH  /v1/nodes/{id}/memory           # 更新节点 Memory（MCP 也有：patch_node_memory）
POST   /v1/nodes/{id}/memory/snapshot  # 创建 Memory 快照（HTTP only）
```

---

## Run

```
POST   /v1/nodes/{id}/runs            # 创建 Run
GET    /v1/nodes/{id}/runs            # Run 列表（默认 summary + cursor）
GET    /v1/runs/{id}                  # Run 详情（默认不带日志）
POST   /v1/runs/{id}/finish           # 结束 Run
POST   /v1/runs/{id}/logs             # 追加 Run 日志
```

### 示例：结束 Run

```
POST /v1/runs/{id}/finish
Content-Type: application/json

{
  "result": "done",
  "output_preview": "迁移完成，3 个表更新成功"
}
```

`result` 合法值：`done`、`canceled`

---

## 产物

```
GET    /v1/tasks/{id}/artifacts            # 任务产物列表
GET    /v1/nodes/{id}/artifacts            # 节点产物列表
POST   /v1/tasks/{id}/artifacts            # 创建链接型产物
POST   /v1/tasks/{id}/artifacts/upload     # 上传文件产物（base64）
GET    /v1/artifacts/{id}/download         # 下载（仅上传型可下载）
```

---

## 全局

```
GET    /healthz                        # 健康检查
GET    /v1/work-items                  # 当前可执行工作项
GET    /v1/search                      # 旧搜索（转发 smart-search）
GET    /v1/smart-search                # 全文搜索（FTS5 + BM25）
GET    /v1/events                      # 全局事件流
POST   /v1/import-plan                 # 导入计划（data 必须是字符串）
POST   /v1/admin/sweep-leases          # 清理过期 lease
POST   /v1/admin/empty-trash           # 清空回收站
POST   /v1/admin/rebuild-index         # 重建 FTS5 全文索引
```

### 示例：全文搜索

```
GET /v1/smart-search?q=用户迁移&scope=node&limit=10
```

| 参数 | 说明 |
|------|------|
| `q` | 搜索关键词（支持 FTS5 语法：AND、OR、NOT、引号） |
| `scope` | `task` / `node` / `memory` / `all`（默认） |
| `task_id` | 限定在某个任务内搜索 |
| `limit` | 结果上限（默认 20，最大 100） |

---

## 常见陷阱速查

| 操作 | 陷阱 | 正确做法 |
|------|------|---------|
| 创建阶段 | 用 `name` 字段 | 字段名是 `title` |
| 导入计划 | `data` 传对象 | `data` 必须是**字符串**；`apply=false` 仅预演 |
| 状态流转 | 传 `status` | 传 `action` |
| 完成节点 | 对 `group` 节点调用 | 只有 `leaf` 节点能完成 |
| retype | 对有子节点的 group 调用 | 只能把**无子节点**的 group 转成 leaf |
| 下载产物 | 对链接型产物调用 | 只有上传型产物可下载 |
| 获取任务树 | 默认请求 | 需显式传 `include_tree=true` |
| 获取 Run 日志 | 默认请求 | 需显式传 `include_logs=true` |
