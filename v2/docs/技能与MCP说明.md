# 技能与 MCP 说明

本文档面向两类用户：

- 使用本地技能文档的 AI 编程助手
- 通过 MCP / DXT 接入本项目的开发者

## 1. 连接信息

| 入口 | 地址 |
|------|------|
| MCP | `http://127.0.0.1:8880/mcp` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| 前端页面 | `http://127.0.0.1:8880` |
| 前端开发 | `http://127.0.0.1:5174/` |

同一个后端 `serve` 进程同时提供 UI、HTTP API 和 MCP。

MCP 连接配置：

```json
{
  "mcpServers": {
    "task-tree": {
      "url": "http://127.0.0.1:8880/mcp"
    }
  }
}
```

## 2. 技能文件

核心技能文档：`./skill/SKILL.md`

覆盖内容：

- 连接信息与数据模型
- 工具速查表（含关键参数）
- 核心工作流与渐进式读取
- 默认读取规则
- 行为规则与 Memory 约定

详细参考文档（按需读取）：

| 文件 | 内容 |
|------|------|
| `skill/docs/task-tree-tools.md` | 完整 MCP 工具清单 + 场景速查索引 |
| `skill/docs/task-tree-api.md` | HTTP API 参考 + 请求/响应示例 |
| `skill/docs/task-tree-best-practices.md` | 最佳实践 + 并发/性能/故障处理 |

## 3. MCP 工具分类

### 项目

`task_tree_list_projects`、`task_tree_get_project`、`task_tree_create_project`、`task_tree_update_project`、`task_tree_delete_project`、`task_tree_project_overview`

### 任务

`task_tree_list_tasks`、`task_tree_get_task`、`task_tree_create_task`（支持 `dry_run`）、`task_tree_update_task`、`task_tree_delete_task`、`task_tree_restore_task`、`task_tree_hard_delete_task`、`task_tree_empty_trash`、`task_tree_transition_task`

### 恢复 / 导航 / 收尾

`task_tree_resume`、`task_tree_next_node`、`task_tree_get_remaining`、`task_tree_get_task_context`、`task_tree_patch_task_context`、`task_tree_wrapup`、`task_tree_get_wrapup`

### 阶段与节点

`task_tree_list_stages`、`task_tree_create_stage`、`task_tree_batch_create_stages`、`task_tree_activate_stage`、`task_tree_create_node`、`task_tree_batch_create_nodes`、`task_tree_list_nodes`、`task_tree_list_nodes_summary`、`task_tree_focus_nodes`、`task_tree_get_node`、`task_tree_get_node_context`、`task_tree_update_node`、`task_tree_reorder_nodes`、`task_tree_move_node`、`task_tree_retype_node`

### 节点执行

`task_tree_claim_and_start_run`、`task_tree_claim`、`task_tree_release`、`task_tree_progress`、`task_tree_complete`、`task_tree_transition_node`、`task_tree_block_node`、`task_tree_patch_node_memory`、`task_tree_get_resume_context`

### Run / Event / Artifact / Search

`task_tree_start_run`、`task_tree_finish_run`、`task_tree_get_run`、`task_tree_list_node_runs`、`task_tree_append_run_log`、`task_tree_list_events`、`task_tree_list_artifacts`、`task_tree_create_artifact`、`task_tree_upload_artifact`、`task_tree_smart_search`、`task_tree_search`、`task_tree_work_items`、`task_tree_tree_view`、`task_tree_import_plan`、`task_tree_sweep_leases`、`task_tree_rebuild_index`

完整清单与参数说明见 [skill/docs/task-tree-tools.md](../skill/docs/task-tree-tools.md)。

短别名：`task_tree.batch_create_nodes`、`task_tree.activate_stage`、`task_tree.list_nodes_summary`

## 4. 默认读取规则

所有读接口默认返回轻量数据：

| 接口 | 默认行为 | 需显式请求 |
|------|---------|-----------|
| `resume` | 轻量包 | `include=events,runs,artifacts,next_node_context,task_memory,stage_memory` |
| `get_task` | 不带树 | `include_tree=true` |
| `list_nodes` | `view_mode=summary` | `view_mode=detail` |
| `get_node_context` | 建议 `preset=summary` | `preset=memory/work/full` |
| `get_run` | 不带日志 | `include_logs=true` |
| `list_node_runs` | `summary + cursor` | `view_mode=detail` |
| `list_artifacts` | `summary + cursor` | `view_mode=detail` |

### 推荐读取顺序

```
恢复现场时：
task_tree_resume
→ task_tree_focus_nodes 或 task_tree_list_nodes_summary
→ task_tree_get_node_context(preset=summary)

普通查询时：
task_tree_next_node / task_tree_focus_nodes / task_tree_list_nodes(parent_node_id=... / subtree_root_node_id=...)
→ task_tree_get_node_context(preset=summary)
→ 按需补充：preset=memory / work，list_node_runs，get_run(include_logs=true)
```

### `resume` 使用约束

- `task_tree_resume` 只用于恢复工作现场，不是默认第一跳。
- 已知 `node_id` 时，优先 `task_tree_get_node` / `task_tree_get_node_context`。
- 只想找下一步时，优先 `task_tree_next_node`。
- 只看局部树或可执行节点时，优先 `task_tree_focus_nodes` / `task_tree_list_nodes` / `task_tree_work_items`。
- 同一轮里对同一 `task_id` 默认最多一次 `resume`。

## 5. HTTP only 能力

以下能力当前没有对应的 MCP 工具：

- Task Memory 原生 PATCH / snapshot（`GET/PATCH /v1/tasks/{id}/memory`，`POST .../snapshot`）
- Stage Memory 原生 PATCH / snapshot（`GET/PATCH /v1/stages/{id}/memory`，`POST .../snapshot`）
- Node Memory snapshot（`POST /v1/nodes/{id}/memory/snapshot`）

已有 MCP 的相关能力：

- 节点 Memory 结构化 patch：`task_tree_patch_node_memory`
- 任务上下文快照读写：`task_tree_get_task_context` / `task_tree_patch_task_context`

## 6. 行为规则（核心）

1. **执行优先**：节点有动词就执行，不要用报告替代
2. **执行前 Claim**：先 `claim_and_start_run` 再执行
3. **完成用 Complete**：`progress(1.0)` 不等于完成
4. **先搜索再开始**：新工作前优先 `smart_search`
5. **渐进式读取**：先判断是否需要 `resume`；只有恢复现场才 `resume`，否则直接最小读取

## 7. DXT

DXT 目录：`./task-tree-dxt/`

- `proxy.mjs`：stdio JSON-RPC → `http://127.0.0.1:8880/mcp` 转发
- `task-tree.dxt`：打包后的安装文件

## 8. Streamable HTTP MCP

`/mcp` 支持：

- `POST /mcp`、`GET /mcp`、`DELETE /mcp`
- `Mcp-Session-Id` 会话管理
- SSE 推送 / 回放 / 恢复流
- 仅接受回环来源

## 9. 文档同步约定

修改 MCP 工具、默认规则、技能工作流或能力边界后，同步更新：

- `./skill/SKILL.md` + `./skill/docs/` 下的文档
- `./docs/技能与MCP说明.md`（本文件）
- `./backend/docs/http-mcp-parity.md`
- `./backend/docs/mcp-open-manifest.txt`

并同步到全局目录：

- `C:\Users\Administrator\.claude\skills\task-tree\`
- `C:\Users\Administrator\.codex\skills\task-tree\`
