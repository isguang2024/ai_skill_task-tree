# 技能与 MCP 说明

本文档面向两类用户：

- 使用本地技能文档的人
- 通过 MCP / DXT 接入 `task-tree-v2` 的人

## 1. V2 对应的技能文件

V2 的技能文件在：

- `C:\Users\Administrator\Desktop\Ai任务步骤记录\task-tree-v2\skill\SKILL.md`

它描述的是：

- 连接信息与端口约定
- MCP 工具清单
- HTTP API 端点
- 常用工作流

当前 V2 技能默认连接：

```json
{
  "mcpServers": {
    "task-tree-v2": {
      "url": "http://127.0.0.1:8880/mcp"
    }
  }
}
```

## 2. V2 的 MCP 地址

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
- `task_tree_create_task`
- `task_tree_update_task`
- `task_tree_delete_task`
- `task_tree_restore_task`
- `task_tree_hard_delete_task`
- `task_tree_empty_trash`
- `task_tree_transition_task`

### 阶段与节点

- `task_tree_create_node`
- `task_tree_list_nodes`
- `task_tree_list_nodes_summary`
- `task_tree_focus_nodes`
- `task_tree_get_node`
- `task_tree_update_node`
- `task_tree_reorder_nodes`
- `task_tree_move_node`
- `task_tree_progress`
- `task_tree_complete`
- `task_tree_block_node`
- `task_tree_claim`
- `task_tree_release`
- `task_tree_retype_node`
- `task_tree_transition_node`

### 阶段

- `task_tree_list_stages`
- `task_tree_create_stage`
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
- `task_tree_get_remaining`
- `task_tree_get_resume_context`
- `task_tree_list_events`
- `task_tree_search`
- `task_tree_work_items`
- `task_tree_sweep_leases`

### 产物

- `task_tree_list_artifacts`
- `task_tree_create_artifact`
- `task_tree_upload_artifact`

完整开放清单见：

- [backend/docs/mcp-open-manifest.txt](/C:/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-v2/backend/docs/mcp-open-manifest.txt)

## 4. 当前哪些能力不是 MCP 工具

下面这些能力在 V2 已经存在，但当前主要通过 HTTP 提供：

- Memory 读写
- 部分 overview/read model 增强字段

也就是说：

- Run 和节点 context 已有 MCP
- Memory 仍以 HTTP 为主
- 前端可以直接用，HTTP 客户端也可以直接用

如果你在写技能或自动化，不要假设这些能力已经能通过 MCP 直接调用。

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

V2 的 DXT 在：

- `C:\Users\Administrator\Desktop\Ai任务步骤记录\task-tree-v2\task-tree-dxt`

其中：

- `proxy.mjs` 负责把 stdio JSON-RPC 转发到 `http://127.0.0.1:8880/mcp`
- `task-tree.dxt` 是打包后的安装文件

## 6. 文档同步约定

如果你改动了技能、MCP 地址、端口、能力边界或导航入口，请同步更新这份文档和项目说明页，保证说明文件始终和最新状态一致。
