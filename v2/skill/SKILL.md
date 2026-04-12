---
name: task-tree
description: Task Tree V2 服务使用指南。涵盖连接信息、快速工作流、工具速查和完整 API 参考。服务地址 `http://127.0.0.1:8880`。
---

# Task Tree V2 — 服务使用指南

## 连接信息

| 入口 | 地址 |
|------|------|
| MCP | `http://127.0.0.1:8880/mcp` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| 前端页面 | `http://127.0.0.1:8880` |
| 健康检查 | `http://127.0.0.1:8880/healthz` |

**MCP 配置（本地）：**
```json
{
  "mcpServers": {
    "task-tree": {
      "url": "http://127.0.0.1:8880/mcp"
    }
  }
}
```

---

## 快速理解

### 数据模型

```
Project → Task → Node(树形) → Run(执行记录)
                  ├→ Stage(阶段)
                  └→ Memory(记忆)
```

- **Project**：项目，用来分组任务
- **Task**：目标任务，包含节点树和执行历史
- **Node**：可执行单元或分组，支持树形结构
- **Stage**：任务划分的阶段（可选），支持状态激活切换
- **Run**：节点执行记录，含日志和结果
- **Memory**：结构化记忆，含备注、摘要、决策、风险等

### 节点状态

```
ready → running → done
  ↓        ↓
paused   blocked → ready (unblock)
  ↓
ready (reopen)
```

---

## 快速工作流

### 【场景1】新建任务并执行

```
1. task_tree_create_task       // 创建任务
2. task_tree_create_node       // 创建节点（可多个）
3. task_tree_claim             // 领取节点执行权
4. task_tree_start_run         // 开始运行
5. task_tree_append_run_log    // 追加日志（重复）
6. task_tree_finish_run        // 结束运行
7. task_tree_complete          // 完成节点
```

### 【场景2】继续上次工作

```
1. task_tree_resume(task_id)           // 获取任务上下文 + 下一节点
2. task_tree_get_node_context(node_id) // 查看节点的Memory和历史
3. task_tree_claim                     // 领取节点
4. [执行工作]
5. task_tree_complete                  // 完成节点
```

### 【场景3】查看待办

```
1. task_tree_work_items     // 获取所有就绪的可执行节点
2. 选择一个 node_id
3. task_tree_claim          // 领取
4. [执行]
```

### 【场景4】进度上报（无需新建 Run）

```
task_tree_progress(node_id, progress: 0.3, message: "进度说明")
// 会自动创建/更新运行记录，父节点进度自动聚合
```

### 【场景5】查看完整树结构

```
1. task_tree_resume(task_id, view_mode: 'summary')  // 获取焦点节点树
   或
2. HTTP GET /v1/tasks/{id}/nodes?limit=10000  // 获取全量节点列表
```

---

## 工具速查表

### 我想…找到任务

| 需求 | 工具 |
|------|------|
| 列出所有任务 | `task_tree_list_tasks` |
| 读取任务详情 | `task_tree_get_task` |
| 搜索任务 | `task_tree_search` |
| 获取任务树+下一步 | `task_tree_resume` ⭐ |
| 获取任务统计 | `task_tree_get_remaining` |

### 我想…操作任务

| 需求 | 工具 |
|------|------|
| 创建任务 | `task_tree_create_task` |
| 更新标题/目标 | `task_tree_update_task` |
| 暂停/重启/取消 | `task_tree_transition_task` |
| 删除任务 | `task_tree_delete_task` |
| 从回收站恢复 | `task_tree_restore_task` |

### 我想…操作节点

| 需求 | 工具 |
|------|------|
| 列出节点 | `task_tree_list_nodes` |
| 读取节点详情 | `task_tree_get_node` |
| 创建节点 | `task_tree_create_node` |
| 更新节点 | `task_tree_update_node` |
| 获取节点完整上下文 | `task_tree_get_node_context` ⭐ |
| 获取焦点节点 | `task_tree_focus_nodes` |
| 移动/排序节点 | `task_tree_move_node` / `task_tree_reorder_nodes` |

### 我想…执行和记录

| 需求 | 工具 |
|------|------|
| 领取节点 | `task_tree_claim` |
| 释放节点 | `task_tree_release` |
| 上报进度 | `task_tree_progress` |
| 开始运行 | `task_tree_start_run` |
| 追加运行日志 | `task_tree_append_run_log` |
| 结束运行 | `task_tree_finish_run` |
| 完成节点 | `task_tree_complete` |
| 标记阻塞 | `task_tree_block_node` |
| 切换节点状态 | `task_tree_transition_node` |

### 我想…管理阶段

| 需求 | 工具 |
|------|------|
| 列出阶段 | `task_tree_list_stages` |
| 创建阶段 | `task_tree_create_stage` |
| 激活阶段 | `task_tree_activate_stage` |

### 我想…读取Memory

| 需求 | 工具 |
|------|------|
| 任务 Memory | `HTTP GET /v1/tasks/{id}/memory` |
| 阶段 Memory | `HTTP GET /v1/stages/{id}/memory` |
| 节点 Memory | `HTTP GET /v1/nodes/{id}/memory` |
| 更新 Memory | `HTTP PATCH /v1/[tasks\|stages\|nodes]/{id}/memory` |

### 我想…查看事件和产物

| 需求 | 工具 |
|------|------|
| 任务事件流 | `task_tree_list_events` |
| 产物列表 | `task_tree_list_artifacts` |
| 创建产物 | `task_tree_create_artifact` |
| 上传产物 | `task_tree_upload_artifact` |

---

## Memory 使用约定

### 结构

```json
{
  "manual_note_text": "人工备注",
  "summary_text": "系统摘要",
  "conclusions": ["结论1", "结论2"],
  "decisions": ["决策1"],
  "risks": ["风险1"],
  "blockers": ["阻塞项1"],
  "next_actions": ["下一步1"],
  "evidence": ["证据链接1"]
}
```

### 约定

- `manual_note_text`：用户手动写，存储临时说明、背景、偏好
- `summary_text`：AI/系统生成，高层摘要
- `conclusions`, `decisions`, `risks`, `blockers`, `next_actions`, `evidence`：AI/系统主动填充

### 推荐刷新时机

1. 节点完成后，刷新节点 Memory
2. 阶段切换或阶段完成度变化，刷新阶段 Memory
3. 任务收尾、方向变化或形成关键结论，刷新任务 Memory

---

## 完整工具清单

### 项目管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_projects` | 列出所有项目 |
| `task_tree_create_project` | 创建项目 |
| `task_tree_get_project` | 获取项目详情 |
| `task_tree_update_project` | 更新项目 |
| `task_tree_delete_project` | 删除项目 |
| `task_tree_project_overview` | 项目概览（含任务统计） |

### 任务管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_tasks` | 列出任务（支持 project_id 筛选） |
| `task_tree_create_task` | 创建任务（可含初始节点树） |
| `task_tree_get_task` | 获取任务详情 |
| `task_tree_update_task` | 更新任务 |
| `task_tree_delete_task` | 软删除任务（进回收站） |
| `task_tree_hard_delete_task` | 硬删除任务 |
| `task_tree_restore_task` | 从回收站恢复 |
| `task_tree_transition_task` | 任务状态流转（pause/reopen/cancel） |
| `task_tree_resume` | **核心**：获取任务完整上下文（树、Memory、下一步、事件） |
| `task_tree_get_remaining` | 获取任务剩余进度统计 |

### 节点操作

| 工具 | 说明 |
|------|------|
| `task_tree_list_nodes` | 列出任务的所有节点 |
| `task_tree_list_nodes_summary` | 节点摘要列表（轻量） |
| `task_tree_create_node` | 创建节点（指定 parent_node_id 构建树） |
| `task_tree_get_node` | 获取节点详情 |
| `task_tree_update_node` | 更新节点标题/指令/验收标准 |
| `task_tree_get_node_context` | **核心**：节点完整上下文（祖先、Memory、运行、事件） |
| `task_tree_focus_nodes` | 获取可执行的焦点节点 |
| `task_tree_move_node` | 移动节点到新父节点 |
| `task_tree_reorder_nodes` | 重新排序同级节点 |
| `task_tree_retype_node` | 节点类型转换（group ↔ leaf） |

### 节点执行层

| 工具 | 说明 |
|------|------|
| `task_tree_claim` | 领取节点执行权（设置 lease） |
| `task_tree_release` | 释放节点执行权 |
| `task_tree_progress` | 上报进度（0-1 的增量） |
| `task_tree_complete` | 完成节点（需提供 message） |
| `task_tree_block_node` | 标记节点阻塞（需提供 reason） |
| `task_tree_transition_node` | 节点状态流转（pause/reopen/unblock/cancel） |

### 运行层

| 工具 | 说明 |
|------|------|
| `task_tree_start_run` | 为节点创建运行记录 |
| `task_tree_append_run_log` | 追加运行日志（kind: info/success/error/...） |
| `task_tree_finish_run` | 结束运行（status: success/error, result: passed/done/...） |
| `task_tree_get_run` | 获取运行详情（含完整日志） |
| `task_tree_list_node_runs` | 列出节点的运行历史 |

### 阶段管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_stages` | 列出任务的所有阶段 |
| `task_tree_create_stage` | 创建阶段（可选立即激活） |
| `task_tree_activate_stage` | 激活指定阶段 |

### 其他

| 工具 | 说明 |
|------|------|
| `task_tree_work_items` | 获取当前可执行的工作项 |
| `task_tree_list_events` | 列出任务事件流 |
| `task_tree_search` | 搜索任务/节点 |
| `task_tree_create_artifact` | 创建产物链接 |
| `task_tree_upload_artifact` | 上传产物文件 |
| `task_tree_list_artifacts` | 列出产物 |
| `task_tree_sweep_leases` | 清理过期 lease |
| `task_tree_empty_trash` | 清空回收站 |

---

## 完整 HTTP API

### 项目

```
GET    /v1/projects                       — 列出项目
POST   /v1/projects                       — 创建项目
GET    /v1/projects/{id}                  — 项目详情
PATCH  /v1/projects/{id}                  — 更新项目
DELETE /v1/projects/{id}                  — 删除项目
GET    /v1/projects/{id}/overview         — 项目概览
GET    /v1/projects/{id}/tasks            — 项目下的任务列表
```

### 任务

```
GET    /v1/tasks                          — 列出任务
POST   /v1/tasks                          — 创建任务
GET    /v1/tasks/{id}                     — 任务详情
PATCH  /v1/tasks/{id}                     — 更新任务
DELETE /v1/tasks/{id}                     — 软删除
POST   /v1/tasks/{id}/hard                — 硬删除
POST   /v1/tasks/{id}/restore             — 恢复
POST   /v1/tasks/{id}/transition          — 状态流转
GET    /v1/tasks/{id}/resume              — 任务上下文（核心）
GET    /v1/tasks/{id}/remaining           — 剩余进度
GET    /v1/tasks/{id}/events/stream       — SSE 事件流
```

### 任务 Memory

```
GET    /v1/tasks/{id}/memory              — 获取
PATCH  /v1/tasks/{id}/memory              — 更新
POST   /v1/tasks/{id}/memory/snapshot     — 快照
```

### 阶段

```
GET    /v1/tasks/{id}/stages              — 列出阶段
POST   /v1/tasks/{id}/stages              — 创建阶段
POST   /v1/tasks/{id}/stages/{sid}/activate — 激活阶段
GET    /v1/stages/{id}/memory             — 阶段 Memory
PATCH  /v1/stages/{id}/memory             — 更新阶段 Memory
POST   /v1/stages/{id}/memory/snapshot    — 快照阶段 Memory
```

### 节点

```
GET    /v1/tasks/{id}/nodes               — 列出节点
POST   /v1/tasks/{id}/nodes               — 创建节点
POST   /v1/tasks/{id}/reorder             — 重排序
GET    /v1/nodes/{id}                     — 节点详情
PATCH  /v1/nodes/{id}                     — 更新节点
GET    /v1/nodes/{id}/context             — 节点上下文（核心）
POST   /v1/nodes/{id}/transition          — 状态流转
POST   /v1/nodes/{id}/move                — 移动节点
POST   /v1/nodes/{id}/progress            — 上报进度
POST   /v1/nodes/{id}/complete            — 完成节点
POST   /v1/nodes/{id}/block               — 标记阻塞
POST   /v1/nodes/{id}/claim               — 领取
POST   /v1/nodes/{id}/release             — 释放
POST   /v1/nodes/{id}/retype              — 类型转换
```

### 节点 Memory

```
GET    /v1/nodes/{id}/memory              — 获取
PATCH  /v1/nodes/{id}/memory              — 更新
POST   /v1/nodes/{id}/memory/snapshot     — 快照
```

### 运行

```
POST   /v1/nodes/{id}/runs                — 启动运行
GET    /v1/nodes/{id}/runs                — 列出运行
GET    /v1/runs/{id}                      — 运行详情
POST   /v1/runs/{id}/finish               — 结束运行
POST   /v1/runs/{id}/logs                 — 追加日志
```

### 产物

```
GET    /v1/tasks/{id}/artifacts           — 任务产物
GET    /v1/nodes/{id}/artifacts           — 节点产物
POST   /v1/tasks/{id}/artifacts           — 创建产物
POST   /v1/tasks/{id}/artifacts/upload    — 上传产物
GET    /v1/artifacts/{id}/download        — 下载产物
```

### 全局

```
GET    /v1/work-items                     — 待执行工作项
GET    /v1/search                         — 搜索
GET    /v1/events                         — 事件列表
POST   /v1/admin/sweep-leases             — 清理过期 lease
POST   /v1/admin/empty-trash              — 清空回收站
```

---

## 使用规范

### ⚠️ 执行优先原则（最重要）

**节点要求你做什么，你就去做什么。不要用"输出报告"替代"实际执行"。**

**❌ 错误做法：** 节点说"删除无用文件"，你生成一份"待删除文件清单.md"然后标记完成
```
节点："清理重复的功能实现"
错误行为：写一份 docs/reports/duplicate-audit.md，罗列发现，然后 complete
问题：什么都没改，产出了一堆中间报告，代码原封不动
```

**✅ 正确做法：** 真正去删文件、改代码、跑验证，然后把结果记到 Memory
```
节点："清理重复的功能实现"
正确行为：
1. 分析哪些重复 → 直接删除/合并代码
2. 跑 build/test 验证不破坏功能
3. 在 Memory 的 evidence 中记录改了哪些文件
4. complete 时写清实际改动
```

**判断标准：**
- 节点描述中有动词（删除/迁移/重构/实现/修改）→ **必须执行对应操作**
- 只有明确标注"调研/分析/设计"的节点 → 才可以仅输出文档
- 如果不确定 → 默认执行，不要默认只输出

**产出文件规则：**
- 不要每个节点都生成一份 `.md` 报告
- 审计/分析类结果直接写进 Memory 的结构化字段（evidence、conclusions、decisions）
- 只有最终汇总文档才需要产出文件

---

### 自主推进规则

**完成一个节点后，AI 应自主判断是否继续推进下一个节点。**

**推进判断流程：**
```
节点 complete 后 →
  1. 当前阶段还有 ready 节点？
     ├─ 有 → 自动 claim 下一个，继续执行
     └─ 没有 → 当前阶段已全部完成
  2. 下一个节点是否有前置依赖？
     ├─ 有未完成的依赖 → 跳过，找其他可执行节点
     └─ 无依赖或依赖已完成 → 继续
  3. 下一个节点涉及不可逆操作（删除/迁移）？
     ├─ 是 → 暂停，向用户确认后再执行
     └─ 否 → 直接执行
  4. 阶段全部完成？
     └─ 向用户报告阶段完成，询问是否进入下一阶段
```

**不需要用户确认就可以继续的情况：**
- 同阶段内的连续节点
- 纯代码修改、构建、测试类节点
- 节点之间有明确的顺序关系

**必须暂停确认的情况：**
- 跨阶段切换
- 涉及不可逆操作（删除文件/数据库迁移）
- 节点完成后发现方向可能有误

---

### 执行前必须 Claim

**❌ 错误做法：** 直接开始执行和记录，不领取节点
```
不推荐：直接调用 task_tree_start_run 或 task_tree_progress
问题：其他 AI 不知道你在做，可能重复工作或数据冲突
```

**✅ 正确做法：** 先领取，再执行
```
1. task_tree_claim(node_id)           // 领取节点（设置 lease）
2. task_tree_start_run 或 progress    // 再开始记录
3. task_tree_complete 或 transition   // 完成或变更状态
4. task_tree_release                  // 必要时释放 lease
```

**为什么？** 
- Claim 建立了"这个节点正在被执行"的事实
- 避免多个 AI 同时执行同一个节点
- 其他任务可以看到节点被锁定，不会误操作

---

### 重要操作记录规范

**进度更新（小步进）：**
```
task_tree_progress(node_id, progress: 0.2, message: "完成表单设计")
```
- 当完成了一个中等步骤时更新
- 频率：每 30 分钟或完成一个子任务时
- 会自动创建/更新运行记录

**运行日志（重要事件）：**
```
task_tree_append_run_log(run_id, {
  kind: "info",     // info | success | error | warning
  content: "关键发现：API 响应时间超过预期，需要优化"
})
```
- 记录关键发现、决策点、错误和解决方案
- 频率：每个重大步骤或遇到问题时
- 形成可追溯的执行日志

**Memory 补充（经验沉淀）：**
```
HTTP PATCH /v1/nodes/{id}/memory {
  "summary_text": "完成了 XX 功能：删除 3 个重复文件，合并 2 个工具函数，构建通过",
  "conclusions": ["utils/format.ts 和 helpers/format.ts 完全重复，保留 utils 版本", "iconify-loader.ts 无任何引用，安全删除"],
  "decisions": ["选择保留 utils/ 下的版本，因为引用更多（12 vs 3）", "暂不合并 auth 壳层，需等路由重构完成"],
  "risks": ["删除 iconify-loader 后如果有动态引用可能报错"],
  "blockers": [],
  "next_actions": ["下一节点处理 types 目录清理"],
  "evidence": ["git diff: 删除 frontend/src/helpers/format.ts", "pnpm build 通过，无报错"]
}
```

**Memory 详细度要求：**
- **`summary_text`**：必须写清"做了什么 + 量化结果"，不要只写"已完成"
- **`conclusions`**：记录分析结论和判断依据，方便后续查阅
- **`decisions`**：记录关键决策和选择原因（"选了 A 不选 B，因为…"）
- **`evidence`**：记录实际改动的文件路径、命令输出、验证结果
- **`risks`** / **`blockers`**：诚实记录风险和阻塞项，不要省略
- **`next_actions`**：为下一个执行者留下明确的行动指引

**❌ 不合格的 Memory：**
```json
{ "summary_text": "按清单完成安全删除", "evidence": ["按清单完成安全删除"] }
```

**✅ 合格的 Memory：**
```json
{
  "summary_text": "删除 Instructions/ 目录（5个文件）和 iconify-loader.ts，pnpm build + vue-tsc 通过",
  "conclusions": ["Instructions/ 全目录无引用，属于历史遗留", "iconify-loader.ts 在 vite.config 中已被注释"],
  "decisions": ["直接 rm -rf 而非逐个删除，因为整个目录都是废弃的"],
  "evidence": ["git status: 6 files deleted", "pnpm build exit 0", "vue-tsc --noEmit exit 0"],
  "next_actions": ["继续节点 1.3 清理重复功能"]
}
```

---

### 完成节点的标志

**❌ 错误：** 只更新进度到 1.0 就认为完成
```
task_tree_progress(node_id, progress: 1.0)  // 不够
```

**✅ 正确：** 显式调用 complete
```
task_tree_complete(node_id, {
  message: "表单验证已实现，包含前端和后端验证"
})
```

**区别：**
- `progress: 1.0` — 表示工作量完成，但状态仍是 running
- `task_tree_complete` — 明确标记节点状态为 done，生成完成事件，解锁下一节点

---

### 遇到阻塞的处理

**发现无法继续时：**
```
1. task_tree_block_node(node_id, reason: "依赖节点还未完成")
2. 在 Memory 中记录：blockers: ["依赖节点 XXX"]
3. 释放 lease：task_tree_release(node_id)
```

**解除阻塞时：**
```
task_tree_transition_node(node_id, action: "unblock")
```

---

## 节点设计原则

### 粒度要求

**不推荐：** 一个节点完成多个任务
```
❌ "实现用户登录功能"
   └─ [太宽泛] 包含表单设计、验证、API 调用、错误处理
```

**推荐：** 细化为独立的子节点
```
✅ "用户登录功能"
   ├─ 设计登录表单 UI
   ├─ 实现前端表单验证逻辑
   ├─ 调用后端登录 API
   ├─ 处理登录错误和异常
   └─ 编写登录流程测试
```

### 设计原则

1. **单一职责**：一个节点只负责一个清晰的功能或工作流步骤
2. **可验收**：节点的完成有明确的验收标准，可以用 "是/否" 判断
3. **估时准确**：节点的工作量可以准确估算（通常 1-4 小时）
4. **顺序清晰**：节点之间的依赖关系明确
5. **便于追踪**：Memory 可以记录该节点的关键决策和遇到的问题

### 何时追加子节点

- 任务方向调整或新增需求
- 原有节点范围扩大，超出预期
- 遇到预期外的复杂性或阻塞
- 需要停下来补充某个相关但独立的工作

追加时保持树形结构完整，不要平铺节点。

---

## 典型决策树

**我想完成一个任务，应该用哪些工具？**

```
1. 任务已存在?
   ├─ 是 → task_tree_resume(task_id)
   └─ 否 → task_tree_create_task + task_tree_create_node

2. 获得下一节点后 →  task_tree_claim

3. 有执行文本日志?
   ├─ 是 → task_tree_start_run + task_tree_append_run_log(多个) + task_tree_finish_run
   └─ 否 → 直接 task_tree_progress 上报进度

4. 节点完成 → task_tree_complete

5. 需要补充说明?
   ├─ 是 → HTTP PATCH /v1/nodes/{id}/memory
   └─ 否 → 完成
```
