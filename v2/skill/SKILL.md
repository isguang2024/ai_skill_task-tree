---
name: task-tree
description: Task Tree V2 — AI 任务管理系统。将复杂工作拆解为树形节点，逐个领取执行，通过 Memory 跨会话传递上下文。连接 `http://127.0.0.1:8880`。
---

# Task Tree V2 — AI 任务管理系统

## 这个系统是什么

Task Tree 是一个**面向 AI 的任务管理与知识沉淀系统**。它解决的核心问题是：

- **复杂任务拆解**：把大目标拆成树形节点（支持多层子节点），每个节点单一职责、可独立执行
- **跨会话上下文传递**：AI 的对话窗口有限，但 Memory 和 execution_log 可以无限累积，下次 resume 时完整调回
- **执行追踪与协作**：节点有 claim/release 机制，防止多个 AI 重复执行同一工作
- **知识沉淀**：每个节点完成后，结论、决策、证据都结构化存储，形成可检索的知识库

**典型使用场景**：大型重构、多阶段迁移、跨模块改动、需要多轮对话完成的复杂工程任务。

## 连接信息

| 入口 | 地址 |
|------|------|
| MCP | `http://127.0.0.1:8880/mcp` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| 前端页面 | `http://127.0.0.1:8880` |

## 数据模型

```
Project → Task → Stage(阶段，可选)
                   └→ Node(树形，可多层嵌套)
                        ├→ Run(执行记录 + 日志)
                        └→ Memory(结构化记忆)
```

- **Node 是核心**：支持无限层级的树形结构（group 节点可包含子节点，leaf 节点是最小执行单元）
- **依赖声明**：支持 `depends_on`（node_id）和 `depends_on_keys`（创建期更友好）
- **节点状态**：ready → running → done（可 block/pause/reopen）

## 快速工作流

### 场景1：新建任务并执行

```
1. task_tree_create_task(stages + nodes 一步到位)
   └ nodes 支持树形：parent_node_id 指定父节点，`depends_on/depends_on_keys` 设置依赖
2. task_tree_claim_and_start_run(node_id)       // 领取+开始 2→1
3. [执行实际操作]
4. task_tree_complete(node_id, memory: {...})    // 完成+Memory+next 一步到位
5. 从 .next 获取下一节点，回到步骤 2
```

创建复杂任务前可先预演：

```
task_tree_create_task(..., dry_run: true)
→ 校验树结构和依赖解析，不落库
```

### 场景2：继续上次工作

```
1. task_tree_resume(task_id)          // 获取上下文 + recommended_action
2. 按 recommended_action 执行：
   - claim → task_tree_claim_and_start_run(node_id)
   - continue → 直接继续
   - all_done → 任务收尾
3. [执行] → task_tree_complete(memory: {...})
```

### 场景3：执行较长节点时动态更新

```
1. claim_and_start_run(node_id)
2. [执行一部分工作]
3. progress(node_id, progress: 0.3, message: "完成数据层改造")
4. patch_node_memory(node_id, append_execution_log: "1. 迁移 user 表完成\n2. repository 接口已更新")
5. [继续执行]
6. progress(node_id, progress: 0.7, message: "API 层改造中")
7. patch_node_memory(node_id, append_execution_log: "3. CRUD 接口重写完成\n4. 发现 auth 中间件需要适配")
8. [完成]
9. complete(node_id, memory: { summary_text: "...", conclusions: [...] })
   → complete 时用 execution_log（覆盖模式）写入最终完整版，或用 append_execution_log 追加最后一段
```

关键点：
- append_execution_log 是追加模式，自动在现有内容末尾换行追加
- execution_log 是覆盖模式，替换整个字段
- AI 应根据情况选择：执行中用追加，完成时可用覆盖写最终版
- 每完成一个重要步骤就更新一次，不要等到最后才写

### 场景4：查看待办

```
task_tree_work_items → 选择 node_id → task_tree_claim_and_start_run
```

### 场景4：搜索历史知识

```
task_tree_smart_search(q: "关键词")
→ 全文检索标题、instruction、Memory、execution_log
→ FTS5 + BM25 排序，返回最相关的节点和任务
```

### 场景5：回读历史上下文

```
task_tree_get_node_context(node_id)     → 完整上下文（祖先、Memory、运行、事件）
task_tree_get_node(id, include_context) → 节点详情 + 可选上下文
task_tree_list_node_runs → task_tree_get_run  → 原始运行日志（最细粒度）
```

## 推荐工具

带 [合并] 标记的是优化后的合并工具，减少 MCP 调用次数。

| 需求 | 工具 | 说明 |
|------|------|------|
| 创建任务 | create_task | 支持 stages + nodes 一步到位 |
| 创建预演 | create_task(dry_run=true) | 仅校验并返回 preview_tree/resolved_dependencies |
| 批量创建节点 | batch_create_nodes [合并] | N 次创建合为 1 次调用 |
| 批量创建阶段 | batch_create_stages | 阶段批量创建，事务原子 |
| 领取+开始 | claim_and_start_run [合并] | claim + start_run 合为 1 次 |
| 完成节点 | complete [合并] | 自动 finish run + 写 Memory + 返回 next_node；支持 result_payload |
| 下一节点推荐 | next_node | 返回 recommended_action + alternatives（并行候选） |
| 上报进度 | progress | 支持 log_content 内联日志 |
| 获取任务上下文 | resume | 含树、Memory、下一步、推荐动作 |
| 任务上下文快照 | get/patch_task_context | 读写 architecture_decisions/reference_files/context_doc_text |
| 树形可视化 | tree_view | 输出缩进树，快速核对层级与依赖 |
| 结构化导入 | import_plan | 支持 Markdown/YAML/JSON，dry-run 或 apply |
| 获取节点上下文 | get_node 或 get_node_context | get_node 可传 include_context=true |
| 全文搜索 | smart_search | FTS5 + BM25，搜标题、指令、Memory、日志 |
| 列出节点 | list_nodes | view_mode: slim / summary / detail / events |
| 更新 Memory | patch_node_memory | 支持 append_execution_log 追加和 execution_log 覆盖两种模式 |
| 状态流转 | transition_node | block / pause / reopen / unblock / cancel |

完整工具清单见 docs/task-tree-tools.md，HTTP API 见 docs/task-tree-api.md

## 行为规则

### 1. 执行优先（最重要）

**节点要求你做什么，就去做什么。不要用"输出报告"替代"实际执行"。**

- 节点有动词（删除/迁移/重构/实现）→ 必须执行
- 只有标注"调研/分析/设计"的节点 → 才可仅输出文档
- 审计/分析结果写进 Memory 结构化字段，不要生成 .md 报告

### 2. 任务拆解要充分

**创建任务时，优先拆成多层子节点，而不是平铺一堆同级节点。**

```
✅ 推荐：树形结构
"用户模块重构"
├─ "数据层改造"
│  ├─ "迁移 user 表结构"
│  ├─ "更新 repository 接口"
│  └─ "补充迁移测试"
├─ "API 层改造"
│  ├─ "重写 user CRUD 接口"
│  └─ "更新 OpenAPI spec"
└─ "前端适配"
   ├─ "更新用户列表页"
   └─ "更新用户详情页"

❌ 不推荐：平铺
"迁移 user 表" / "更新 repository" / "补充测试" / "重写接口" / ...（全部同级）
```

- 每个 leaf 节点应该是 1-4 小时可完成的单一任务
- group 节点用于组织结构，其进度由子节点自动聚合
- 用 depends_on 声明依赖关系（如"API 层"依赖"数据层"完成）
- 批量创建时优先用 depends_on_keys（创建期不依赖 node_id）
- 用 batch_create_nodes 一次性创建多个节点，减少 MCP 调用

### checkpoint 节点约定

- 不新增 kind，采用 `kind=leaf + role=checkpoint`
- 在 `metadata.checkpoint_spec.required_commands` 声明门禁命令
- complete 时必须在 `result_payload.commands_verified` 覆盖这些命令，否则会 409 拒绝完成

### 3. 执行前必须 Claim

先 claim_and_start_run，然后执行，最后 complete。不要跳过 claim 直接执行。

### 4. 完成 = complete，不是 progress(1.0)

progress(1.0) 状态仍是 running。必须调 complete 才标记 done、解锁下一节点。

### 5. 自主推进与节点选择

recommended_action 是系统建议，不是强制指令。AI 应该自主判断执行顺序：

```
节点 complete 后 →
  1. 查看当前 group 下的子节点列表（list_nodes 或 resume 返回的树）
  2. 自主判断下一个执行哪个：
     - 优先执行依赖已满足的 ready 节点
     - 如果有多个 ready 节点，按逻辑顺序选择（如先数据层再 API 层）
     - 如果 recommended_action 的建议合理就采纳，不合理就忽略
  3. 涉及不可逆操作？ → 暂停确认
  4. 阶段全部完成？ → 报告用户，询问是否进入下一阶段
```

AI 可以做的自主判断：
- 浏览某个 group 节点的子节点，决定先做哪个
- 发现节点之间有隐含依赖（文档没声明但逻辑上存在），调整执行顺序
- 跳过当前推荐的节点，选择更紧急或更合理的节点
- 发现节点描述不清晰，先查看其子节点和上下文再决定

### 6. 遇到阻塞

```
transition_node(action: "block", message: "原因") → Memory 记录 blockers → release
```

## 检索与知识利用

Task Tree 不仅是任务管理，也是**可检索的知识库**。善用检索能力：

| 场景 | 做法 |
|------|------|
| 不确定之前是否做过类似工作 | smart_search 搜索历史节点 |
| 需要回顾之前的决策原因 | get_node_context 查看 Memory 中的 decisions |
| 需要了解某阶段的整体情况 | get_node_context 传入 stage_node_id，查看阶段聚合摘要 |
| 需要查看具体执行细节 | list_node_runs 然后 get_run 查看原始日志 |
| 想知道任务还剩多少工作 | get_remaining 查看统计 |
| 想看所有待办 | work_items 或 focus_nodes |

**重要**：在开始新工作前，先搜索是否有相关的历史节点。已完成节点的 Memory 中可能包含有价值的结论、踩过的坑、做过的决策，避免重复劳动。

## Memory 约定

### 结构

```json
{
  "manual_note_text": "人工备注（用户手动写）",
  "summary_text": "做了什么 + 量化结果",
  "execution_log": "详细过程：改了哪些文件、命令、问题、解决方案",
  "conclusions": ["结论和判断依据"],
  "decisions": ["选了A不选B，因为…"],
  "risks": ["已知风险"],
  "blockers": ["阻塞项"],
  "next_actions": ["下一步行动指引"],
  "evidence": ["文件路径、命令输出、验证结果"]
}
```

### 详细度要求

- **summary_text**：做了什么 + 量化结果，不要只写"已完成"
- **execution_log**：详细步骤，用换行分隔。这是压缩后的"详细记忆"，下次 resume 可完整调回
- **conclusions/decisions**：判断依据和选择原因，帮助后续执行者理解为什么这样做
- **evidence**：文件路径、命令输出、验证结果
- **risks/blockers**：诚实记录，不要省略

### 刷新时机

1. 节点完成 → 刷新节点 Memory（通过 complete 的 memory 参数一步到位）
2. 阶段完成 → 刷新阶段 Memory
3. 任务收尾/方向变化 → 刷新任务 Memory

## 双层上下文压缩（重要 — 必须严格执行）

对话上下文昂贵，MCP 存储廉价。AI 在执行任务时必须控制自己的输出详略。

### 第一层：节点完成时的输出规则

完成一个节点后，AI 的回复必须遵守以下格式：

1. 先调用 complete，把详细信息写入 memory 参数（summary_text、execution_log、conclusions、decisions、evidence）
2. 然后在对话中只输出极简摘要，格式如下：

```
节点 X.X「标题」已完成：
- 结果：[一句话描述做了什么 + 量化结果]
- 下一步：节点 X.X「下一个标题」(node_id)
```

禁止在对话中输出：
- 完整的命令输出或日志
- 文件的完整内容
- 详细的推理过程或调试记录
- 这些内容应该写入 memory 的 execution_log 字段

对比示例：

```
错误（占用大量上下文）：
  我完成了节点 1.2，具体操作如下：
  1. 首先我执行了 grep -r "oldFunction" 找到了 15 处引用...
  2. 然后我逐个修改了以下文件：
     - src/utils/format.ts 第 23 行...
     - src/helpers/date.ts 第 45 行...
  3. 接着我运行了 pnpm build，输出如下：
     vite v5.2.0 building for production...
     ...（50行构建日志）...
  4. 最后 vue-tsc --noEmit 通过，无报错
  节点已完成，下面继续执行节点 1.3...

正确（极简，详细信息已在 Memory 中）：
  节点 1.2「清理重复功能」已完成：
  - 结果：删除 3 个重复文件，合并 2 个工具函数，build + 类型检查通过
  - 下一步：节点 1.3「梳理 utils 目录」(nd_xxx)
```

### 第二层：阶段完成时的输出规则

当一个阶段的所有节点都完成后，AI 必须：

1. 刷新阶段 Memory
2. 在对话中用一行总结替代该阶段所有节点级摘要：

```
阶段 1「数据库迁移」已完成（stage_node_id=nd_xxx）：迁移 5 张表，全部通过
```

3. 进入新阶段后，在后续回复中不再重复旧阶段的节点级细节
4. 如果需要引用旧阶段的信息，只写"见阶段 X（nd_xxx）"，不要展开

对比示例：

```
错误（进入阶段 3 时仍保留旧阶段细节）：
  之前在阶段 1 中，我们完成了 5 个节点：
  1.1 扫描文件，发现 23 个旧版文件...
  1.2 删除了 15 个文件，包括 xxx、yyy...
  1.3 合并了 format.ts 和 helpers/format.ts...
  ...（大段回顾）
  现在开始阶段 3...

正确：
  阶段 1「代码清理」已完成（nd_abc）：删除 23 文件，合并 4 工具函数
  阶段 2「结构优化」已完成（nd_def）：重组 3 个目录，测试通过
  现在开始阶段 3...
```

### 需要回顾旧阶段细节时

不要凭记忆复述，从 MCP 调回：

```
get_node_context(stage_node_id)  → 阶段聚合摘要
get_node_context(node_id)        → 节点 execution_log
list_node_runs → get_run         → 原始运行日志
```

## 系统能力总览

| 能力 | 说明 |
|------|------|
| 任务创建 | 一步到位创建含阶段和多层节点的完整任务 |
| 树形节点 | 支持无限层级的 group/leaf 节点 |
| 依赖管理 | 声明节点间依赖，自动阻塞和解锁 |
| 执行追踪 | 领取、运行、进度、完成全流程记录 |
| Memory 系统 | 结构化字段，支持节点、阶段、任务三级 |
| 全文检索 | FTS5 + BM25，覆盖标题、指令、Memory、日志 |
| 阶段管理 | 按阶段组织工作，支持激活和切换 |
| 产物管理 | 关联文件和链接到节点或任务 |
| 事件流 | 完整操作事件记录，支持 SSE 实时推送 |
| 批量操作 | 批量创建节点、领取并开始等合并调用 |

## 参考文档

需要详细信息时用 Read 工具读取，路径相对于本文件所在的 skill 目录：

| 文档 | 内容 |
|------|------|
| docs/task-tree-tools.md | 完整 MCP 工具清单，含废弃工具 |
| docs/task-tree-api.md | 完整 HTTP API 参考 |
| docs/task-tree-best-practices.md | 最佳实践、决策树、Memory 示例 |
