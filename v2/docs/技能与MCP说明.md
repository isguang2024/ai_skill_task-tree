# 技能与 MCP 说明

本文档面向两类用户：

- 使用本地技能文档的人
- 通过 MCP / DXT 接入本项目的人

## 1. V2 对应的技能文件

本项目的技能文件在：

- `./skill/SKILL.md`

它描述的是：

- 连接信息与端口约定
- MCP 工具清单
- HTTP API 端点
- 常用工作流

当前技能默认连接：

```json
{
  "mcpServers": {
    "task-tree": {
      "url": "http://127.0.0.1:8880/mcp"
    }
  }
}
```

## 2. 本项目的 MCP 地址

- `http://127.0.0.1:8880/mcp`

同一个后端还会提供：

- UI：`http://127.0.0.1:8880`
- HTTP API：`http://127.0.0.1:8880/v1/...`

## 3. 当前有哪些 MCP 工具

V2 当前 MCP 主要覆盖五大类。

### 项目

- `task_tree_list_projects`
- `task_tree_get_project`
- `task_tree_create_project`
- `task_tree_update_project`
- `task_tree_delete_project`
- `task_tree_project_overview`

### 任务

- `task_tree_list_tasks`
- `task_tree_get_task`
- `task_tree_create_task`（支持 `dry_run` 预演）
- `task_tree_update_task`
- `task_tree_delete_task`
- `task_tree_restore_task`
- `task_tree_hard_delete_task`
- `task_tree_empty_trash`
- `task_tree_transition_task`

### 阶段与节点

- `task_tree_create_node`（支持 `depends_on_keys`）
- `task_tree_batch_create_nodes`（事务原子）
- `task_tree_list_nodes`
- `task_tree_list_nodes_summary`
- `task_tree_focus_nodes`
- `task_tree_get_node`
- `task_tree_update_node`（支持 `depends_on_keys`）
- `task_tree_reorder_nodes`
- `task_tree_move_node`
- `task_tree_progress`
- `task_tree_complete`（支持 `result_payload`，checkpoint 可强校验）
- `task_tree_block_node`
- `task_tree_claim`
- `task_tree_release`
- `task_tree_retype_node`
- `task_tree_transition_node`

### 阶段

- `task_tree_list_stages`
- `task_tree_create_stage`
- `task_tree_batch_create_stages`
- `task_tree_activate_stage`

### Run 与上下文

- `task_tree_start_run`
- `task_tree_finish_run`
- `task_tree_get_run`
- `task_tree_list_node_runs`
- `task_tree_append_run_log`
- `task_tree_get_node_context`

### 上下文与搜索

- `task_tree_resume`
- `task_tree_next_node`
- `task_tree_get_remaining`
- `task_tree_get_resume_context`
- `task_tree_list_events`
- `task_tree_search`
- `task_tree_smart_search`
- `task_tree_work_items`
- `task_tree_sweep_leases`
- `task_tree_tree_view`
- `task_tree_import_plan`
- `task_tree_get_task_context`
- `task_tree_patch_task_context`
- `task_tree_patch_node_memory`
- `task_tree_rebuild_index`

### 收尾与总结

- `task_tree_wrapup`（写入任务收尾总结）
- `task_tree_get_wrapup`（读取任务收尾总结）

### 节点执行

- `task_tree_claim_and_start_run`（领取 + 开始运行，2→1 合并）

### 产物

- `task_tree_list_artifacts`
- `task_tree_create_artifact`
- `task_tree_upload_artifact`

完整开放清单见：

- [backend/docs/mcp-open-manifest.txt](../backend/docs/mcp-open-manifest.txt)

短别名（与旧名并存）：

- `task_tree.batch_create_nodes`
- `task_tree.activate_stage`
- `task_tree.list_nodes_summary`

## 4. 当前哪些能力不是 MCP 工具

下面这些能力在 V2 已经存在，但当前仅通过 HTTP 提供，**没有**对应 MCP 工具：

- Stage/Task 的 memory 原生 patch（manual_note/full patch）—— `PATCH /v1/stages/{id}/memory`、`PATCH /v1/tasks/{id}/memory`
- 节点 memory 手动快照 —— `POST /v1/nodes/{id}/memory/snapshot`
- 部分 overview/read model 增强字段

**已有 MCP 工具的能力**（不再需要 HTTP）：

- 节点 Memory 结构化写入：`task_tree_patch_node_memory`
- Run 创建、日志、结束：`task_tree_start_run` / `task_tree_append_run_log` / `task_tree_finish_run`
- 任务上下文快照读写：`task_tree_get_task_context` / `task_tree_patch_task_context`
- 收尾总结：`task_tree_wrapup` / `task_tree_get_wrapup`

## 4.1 Memory 的推荐使用方式

V2 里的 Memory 不应该只被当成“人工备注框”。

推荐约定是：

- `manual_note_text`：人工填写
- `summary / decisions / risks / next_actions`：AI 或系统主动生成

因此，技能用户在使用 V2 时，应该默认让 AI 在这些时机尝试刷新 Memory：

1. 节点完成后，刷新节点 Memory
2. 阶段切换或阶段完成度明显变化后，刷新阶段 Memory
3. 任务收尾、任务方向变化或形成关键结论后，刷新任务 Memory

当前这条工作流已经加入 V2 技能文档，即使现在部分能力仍通过 HTTP 调用，推荐行为也已经明确下来。

## 5. DXT 对应关系

本项目的 DXT 在：

- `./task-tree-dxt`

其中：

- `proxy.mjs` 负责把 stdio JSON-RPC 转发到 `http://127.0.0.1:8880/mcp`
- `task-tree.dxt` 是打包后的安装文件

## 6. 文档同步约定

如果你改动了技能、MCP 地址、端口、能力边界或导航入口，请同步更新这份文档和项目说明页，保证说明文件始终和最新状态一致。

并且同步规则不只针对 `SKILL.md`：`skill/docs/` 下文档也要一起同步到全局目录。

- 本地源：
  - `./skill/SKILL.md`
  - `./skill/docs/task-tree-api.md`
  - `./skill/docs/task-tree-best-practices.md`
  - `./skill/docs/task-tree-tools.md`
- 全局目标：
  - `C:\Users\Administrator\.claude\skills\task-tree\`
  - `C:\Users\Administrator\.codex\skills\task-tree\`
