---
name: task-tree
description: Task Tree V2 — AI 任务管理系统。将复杂工作拆解为树形节点，逐个领取执行，通过 Memory 跨会话传递上下文。连接 `http://127.0.0.1:8880`。
---

# Task Tree V2 — AI 任务管理技能

## 连接信息

| 入口 | 地址 |
|------|------|
| MCP | `http://127.0.0.1:8880/mcp` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| 前端页面 | `http://127.0.0.1:8880` |

MCP 和 HTTP 共享同一后端，核心读写能力完全对齐。以下能力目前仅 HTTP 提供：

- Task / Stage Memory 原生 PATCH 与 snapshot（`PATCH /v1/tasks/{id}/memory`、`PATCH /v1/stages/{id}/memory`）
- Node Memory snapshot（`POST /v1/nodes/{id}/memory/snapshot`）

> **详细使用说明与约定**见 `./docs/` 目录：
> - `task-tree-tools.md` — MCP 工具完整参考（全部参数、场景速查）
> - `task-tree-best-practices.md` — 最佳实践与决策指南（拆解、执行、Memory、并发）
> - `task-tree-api.md` — HTTP API 参考

## 数据模型

```
Project → Task → Stage（可选）
                   └→ Node（树形，支持多层嵌套）
                        ├→ Run（执行记录 + 日志）
                        └→ Memory（结构化记忆）
```

### group 与 leaf 关系（重要）

节点只有两种 kind：**group**（组织容器）和 **leaf**（执行单元）。

```
规则：
├─ group 节点：只做组织，不能 claim/complete
│   ├─ 可包含 group 子节点（多层嵌套）
│   └─ 可包含 leaf 子节点
├─ leaf 节点：承载实际工作，可 claim → run → complete
│   └─ 不能有子节点（有子节点会自动变为 group）
└─ 创建时不指定 kind → 有 children 自动为 group，否则为 leaf
```

**允许的树结构示例**：

```
✅ group "后端重构"
   ├─ group "数据层"          ← group 嵌套 group
   │   ├─ leaf "迁移表结构"    ← group 下的 leaf
   │   ├─ leaf "更新 DAO"
   │   └─ leaf "补测试"
   ├─ group "API 层"
   │   ├─ leaf "重写接口"
   │   └─ leaf "更新文档"
   └─ leaf "集成测试"          ← group 下直接挂 leaf 也可以
```

**禁止的操作**：

| 操作 | 结果 |
|------|------|
| 对 group 节点调 `claim`/`complete` | 报错：only leaf node |
| 对有子节点的 leaf 调 `complete` | 该节点已自动变为 group，报错 |
| 创建 leaf 后再给它加子节点 | leaf 自动升级为 group |

**批量创建多子节点**（推荐 `batch_create_nodes`）：

```json
{
  "task_id": "tsk_xxx",
  "nodes": [
    {
      "title": "数据层", "kind": "group", "key": "data",
      "children": [
        {"title": "迁移表结构", "kind": "leaf", "key": "migrate"},
        {"title": "更新 DAO", "kind": "leaf", "key": "dao", "depends_on_keys": ["migrate"]},
        {"title": "补测试", "kind": "leaf", "depends_on_keys": ["dao"]}
      ]
    }
  ]
}
```

### 依赖关系

- 创建时用 `depends_on_keys`，落库后自动解析为 `depends_on`
- 依赖未满足的节点状态为 `pending`，满足后变为 `ready`

### 节点状态机

```
ready → running → done
  ↕        ↓       
paused   canceled  
  ↕        
blocked  
```

## 核心工作流

### 1. 新建任务并执行

```
task_tree_create_task(stages + nodes)     # 一步创建完整骨架
→ task_tree_claim_and_start_run(node_id)  # 领取 + 开始
→ 执行步骤 A
  → task_tree_progress(0.3, log_content="步骤A完成：...")  # 每完成一个关键步骤都要上报
→ 执行步骤 B
  → task_tree_progress(0.6, log_content="步骤B完成：...")  # 持续上报，系统自动记录过程
→ 执行步骤 C
  → task_tree_progress(0.9, log_content="步骤C完成：...")
→ task_tree_complete(node_id, memory:{...})  # 完成 + 写结构化 Memory
→ 从 .next 或 task_tree_next_node 获取下一节点
```

> **progress 调用频率**：每完成一个关键操作（写完代码、测试通过、配置修改等）就调一次，不要攒到最后才报。这是执行过程的唯一记录来源。

### 2. 恢复上下文并继续

```
task_tree_resume(task_id)                 # 轻量恢复
→ 按 recommended_action 或局部树选择节点
→ task_tree_claim_and_start_run(node_id)
→ 执行步骤 → progress(0.x, log_content="...") → 执行步骤 → progress → ... → complete
```

### 3. 渐进式读取（先摘要再下钻）

```
task_tree_resume                          # 第1层：任务概览
→ task_tree_focus_nodes                   # 第2层：可执行节点
→ task_tree_get_node_context(preset=summary) # 第3层：节点概要
→ 按需补充：preset=memory / work / full
```

## 全量工具速查表

### 项目管理

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_list_projects` | 列出所有项目 | `q`（关键词过滤） |
| `task_tree_create_project` | 创建项目 | `title`，`key`，`description`，`is_default` |
| `task_tree_get_project` | 项目详情 | `project_id` |
| `task_tree_update_project` | 更新项目 | `project_id`，可改 `title`/`key`/`description`/`is_default` |
| `task_tree_delete_project` | 删除项目 | `project_id` |
| `task_tree_project_overview` | 项目概览 + 任务统计 | `project_id`，`view_mode=summary_with_stats`（推荐） |

### 任务管理

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_list_tasks` | 列出任务 | `status`，`project_id`，`q` |
| `task_tree_create_task` | 创建任务（可含骨架） | `title`，`goal`，`project_id`，`stages`，`nodes`，`dry_run=true` 预演 |
| `task_tree_get_task` | 任务详情 | `task_id`，`include_tree=true` 才返回树 |
| `task_tree_update_task` | 更新任务 | `task_id`，可改 `title`/`task_key`/`goal` |
| `task_tree_delete_task` | 软删除（移入回收站） | `task_id` |
| `task_tree_hard_delete_task` | 硬删除（不可恢复） | `task_id`（必须已在回收站） |
| `task_tree_restore_task` | 从回收站恢复 | `task_id` |
| `task_tree_empty_trash` | 清空回收站 | 无 |
| `task_tree_transition_task` | 任务状态流转 | `task_id`，`action=pause/reopen/cancel` |

### 任务恢复 / 导航 / 收尾

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_resume` | **核心入口**：轻量恢复包 | `task_id`，`include=events,runs,artifacts,next_node_context,task_memory,stage_memory` |
| `task_tree_next_node` | 推荐下一可执行节点 | `task_id` |
| `task_tree_get_remaining` | 剩余统计 | `task_id` |
| `task_tree_get_task_context` | 任务上下文快照 | `task_id` |
| `task_tree_patch_task_context` | 更新任务上下文 | `task_id`，支持部分更新 |
| `task_tree_wrapup` | 写入任务收尾总结 | `task_id`，`summary`，`conclusions` |
| `task_tree_get_wrapup` | 读取任务收尾总结 | `task_id` |

### 阶段管理

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_list_stages` | 列出阶段 | `task_id` |
| `task_tree_create_stage` | 创建阶段 | `task_id`，`title`（**不是** `name`） |
| `task_tree_batch_create_stages` | 批量创建阶段 | `task_id`，`stages` 数组 |
| `task_tree_activate_stage` | 激活阶段 | `task_id`，`stage_node_id` |

### 节点读取

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_list_nodes` | 节点列表（最灵活） | `task_id`，`view_mode`，`filter_mode`，`parent_node_id`，`subtree_root_node_id`，`max_relative_depth`，`cursor` |
| `task_tree_list_nodes_summary` | 轻量节点列表 | `task_id` |
| `task_tree_focus_nodes` | 可执行节点 + 祖先链 | `task_id` |
| `task_tree_get_node` | 单个节点详情 | `node_id` |
| `task_tree_get_node_context` | 节点上下文聚合 | `node_id`，`preset=summary/memory/work/full` |
| `task_tree_get_resume_context` | 节点级 resume 上下文 | `task_id`，`node_id` |

### 节点写入

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_create_node` | 创建单个节点 | `task_id`，`title`，`kind=leaf/group`，`parent_node_id`，`instruction`，`depends_on_keys` |
| `task_tree_batch_create_nodes` | 批量创建（支持多层 `children`） | `task_id`，`nodes` 数组 |
| `task_tree_update_node` | 更新节点 | `node_id`，可改 `title`/`instruction`/`acceptance_criteria`/`estimate` |
| `task_tree_reorder_nodes` | 批量重排同级 | `node_ids` 数组 |
| `task_tree_move_node` | 移动节点 | `node_id`，`target_parent_node_id`，`position` |
| `task_tree_retype_node` | group ↔ leaf 转换 | `node_id`（仅无子节点的 group 可转 leaf） |
| `task_tree_delete_node` | 删除节点及其子节点（软删除） | `node_id`（不能删除 running/claimed 节点） |

### 节点执行

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_claim_and_start_run` | **推荐**：领取 + 开始 | `node_id`，可选 `actor`/`trigger_kind` |
| `task_tree_claim` | 仅领取 lease | `node_id` |
| `task_tree_release` | 释放 lease | `node_id` |
| `task_tree_progress` | **上报进度 + 自动记录过程 + 续租 lease** | `node_id`，`progress`（0-1），`log_content`（写入 run_logs）。每次调用自动续租 15 分钟 |
| `task_tree_complete` | **完成节点** | `node_id`，`memory:{...}` 结构化字段，`result_payload` |
| `task_tree_transition_node` | 状态流转 | `node_id`，`action=block/pause/reopen/cancel/unblock`。对 group 调 cancel 会**级联取消**所有子节点 |
| `task_tree_patch_node_memory` | 更新节点 Memory | `node_id`，结构化字段（execution_log 已废弃） |

### Run / Event / Artifact

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_start_run` | 创建 Run | `node_id`（通常用 `claim_and_start_run` 代替） |
| `task_tree_append_run_log` | 追加 Run 日志 | `run_id`，`kind`，`content` |
| `task_tree_finish_run` | 结束 Run | `run_id`，`status`，`result=done/canceled` |
| `task_tree_list_node_runs` | Run 历史 | `node_id`，`view_mode=summary/detail`，`cursor` |
| `task_tree_get_run` | Run 详情 | `run_id`，`include_logs=true` 才返回日志 |
| `task_tree_list_events` | 事件流 | `task_id` 或 `node_id`，`limit`，`cursor` |
| `task_tree_list_artifacts` | 产物列表 | `task_id` 或 `node_id`，`view_mode`，`cursor` |
| `task_tree_create_artifact` | 创建链接型产物 | `task_id`，`node_id`，`title`，`url`/`content` |
| `task_tree_upload_artifact` | base64 上传文件产物 | `task_id`，`node_id`，`filename`，`content_base64` |

### 搜索 / 全局

| 工具 | 用途 | 关键参数 |
|------|------|---------|
| `task_tree_smart_search` | **推荐**：全文搜索（FTS5 + BM25） | `q`，`scope=task/node/memory`，`task_id`，`limit` |
| `task_tree_work_items` | 当前可执行工作项 | 无参数 |
| `task_tree_tree_view` | 缩进树文本视图 | `task_id`，支持阶段过滤 |
| `task_tree_import_plan` | 导入计划 | `data`（**必须是字符串**），`apply=false` 预演 |
| `task_tree_sweep_leases` | 清理过期 lease | 无 |
| `task_tree_rebuild_index` | 重建 FTS5 索引 | 无 |

### 已废弃

| 旧工具 | 替代 |
|--------|------|
| `task_tree_search` | `task_tree_smart_search` |
| `task_tree_block_node` | `task_tree_transition_node(action=block)` |
| `execution_log` / `append_execution_log` 参数 | `progress(log_content=...)` 自动记录 |

## 默认读取规则（重要）

所有读接口默认返回轻量数据，重字段必须显式请求：

| 接口 | 默认行为 | 需显式打开 |
|------|---------|-----------|
| `resume` | 轻量包（task + tree + remaining） | `include=events,runs,artifacts,next_node_context,task_memory,stage_memory` |
| `get_task` | 不带树 | `include_tree=true` |
| `list_nodes` | `view_mode=summary` | `view_mode=detail` / `events` |
| `get_node_context` | 建议 `preset=summary` | `preset=memory` / `work` / `full` |
| `get_run` | 不带日志 | `include_logs=true` |
| `GET /memory` | 默认 `structured`（无 execution_log） | `?level=full` |

**不要**默认读整树、完整 context、完整 run 日志。

## 行为规则

### 1. 执行优先
节点有动词（删除、迁移、重构、实现）→ **必须实际执行**，不要用文字报告替代。

### 2. 执行前必须 Claim
先 `claim_and_start_run` → 再执行 → 最后 `complete`。不要跳过 claim。

### 3. 执行过程中持续上报 progress
每完成一个关键步骤（写完一段代码、测试通过、完成配置等）就调 `progress(进度值, log_content="做了什么")`。**不要攒到最后才报**——progress 是执行过程的唯一记录来源，跳过就丢失了过程信息。

### 4. 完成必须用 `complete`
`progress(1.0)` 只上报进度到 100%，**不会**自动标 done。只有 `complete` 才标记 done、写 Memory、解锁后续节点。

### 5. group/leaf 严格区分
group 只做组织容器，leaf 承载执行。对 group 调 `claim`/`complete` 会报错。创建多子节点时父节点必须是 group。

### 6. 自主推进但按局部树判断
`recommended_action` 是建议。应在依赖已满足的 `ready` 节点中选择最合理的下一步。

### 7. 产物记录用链接，不用上传
AI 生产的文件（代码、文档等）直接写到文件系统，再用 `create_artifact(uri=路径)` 记录链接。**禁止** AI 生成内容后 base64 编码再 `upload_artifact`——这会双倍消耗 output tokens。`upload_artifact` 仅限非 AI 生成的外部文件（截图、下载的附件等）。

### 8. 先搜索再开始新工作
开始新节点前优先 `task_tree_smart_search`，避免重复劳动。

### 9. 任务拆解要充分
优先拆成多层树（group → leaf），不要平铺。每个 leaf 控制在 1-4 小时。

### 10. 执行中发现新问题：就地扩展任务树
执行节点过程中发现额外问题、优化点或新需求时，**不要忽略，不要等到后面再说**。应立即扩展任务树：

| 场景 | 操作 |
|------|------|
| 当前节点下需要拆分子任务 | 当前节点转为 group（`retype_node`），再 `batch_create_nodes` 添加 leaf 子节点 |
| 发现同级别的新工作 | 在同一 parent 下 `create_node` 新增 leaf/group |
| 发现新的独立工作线 | `create_stage` 新增阶段 + `batch_create_nodes` 添加节点 |
| 现有节点描述需调整 | `update_node` 修改 title/instruction/acceptance_criteria |
| 现有节点粒度太大 | 转 group + 拆子节点；或新增兄弟节点分担 |

**原则**：任务树是活的，随时可以新增节点、拆分子树、开辟新路线。发现问题就记录到树中，确保不遗漏。在 progress 中记录扩展原因。

### 11. 遇到阻塞
```
task_tree_transition_node(action=block, message=原因)
→ task_tree_patch_node_memory(blockers=...)
→ task_tree_release
```

## Memory 约定

### execution_log：系统自动生成，无需手写

`execution_log` **不再由 AI 手写**，系统自动从 run_logs 聚合。AI 只需 `progress(log_content=...)` 记录过程，`preset=full` 时自动格式化。

**节省 output tokens**：AI 无需花 1000-3000 tokens 手写日志。

### AI 只写结构化字段

```json
{
  "summary_text": "做了什么 + 量化结果",
  "conclusions": ["结论"],
  "decisions": ["关键决策及理由"],
  "risks": ["已知风险"],
  "blockers": ["阻塞项"],
  "next_actions": ["下一步行动"],
  "evidence": ["文件路径、命令输出、验证结果"]
}
```

### 执行过程记录

用 `progress(log_content=...)` 上报，系统自动写入 run_logs 并在 `preset=full` 时聚合：

```
[2026-04-14 10:30] (progress) [30%] 分析 store_memory.go，确认需改造 6 个接入点
[2026-04-14 10:35] (progress) [60%] 实现 aggregateNodeRunLogs，go build exit 0
[2026-04-14 10:40] (complete) 完成
```

## 常见陷阱

| 陷阱 | 正确做法 |
|------|---------|
| 创建 stage 时用 `name` 字段 | 字段名是 `title` |
| `import_plan` 的 `data` 传对象 | `data` 必须是**字符串** |
| `transition` 传 `status` | 传 `action` |
| 对 `group` 节点调 `complete` | 只有 `leaf` 能完成 |
| 创建子节点不设父节点 kind | 父节点必须是 `group` |
| 直接读 `preset=full` | 先读 `summary`，按需下钻 |
| 手写 `execution_log` | 用 `progress(log_content=...)` 自动记录 |
| AI 生成内容后 `upload_artifact` base64 上传 | AI 产物写文件 → `create_artifact(uri=路径)` 记链接 |
| `progress(1.0)` 后以为节点已完成 | `progress(1.0)` 不标 done，必须调 `complete` |
| 需要删除多余节点但找不到工具 | 用 `delete_node` 软删除；取消 group 用 `transition_node(cancel)` 级联 |

## 其他约定

### activate_stage 的 git_suggestion
`activate_stage` 返回的 `git_suggestion` 是建议性的分支名（用于 git 工作流），**不是强制操作**。是否执行由用户决定。

### 反馈机制
使用技能过程中遇到**问题、不便、异常或优化建议**，写入项目根目录的 `task-tree-skill-feedback.md` 文件中。格式参考：

```markdown
## 问题标题
**问题**: 描述遇到的问题
**期望**: 期望的行为
**建议**: 优化建议（可选）
```
