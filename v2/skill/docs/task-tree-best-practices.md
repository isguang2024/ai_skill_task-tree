# Task Tree V2 — 最佳实践与决策指南

> 本文件按需查阅，不在每次对话中加载。需要决策参考时再读取。

## 核心原则

### 1. 先摘要，再下钻

固定读取路径：

```
resume → focus_nodes → get_node_context(summary) → 按需补 memory/work/runs/events
```

**不要**默认读整树、完整 context、完整 run 日志。每一层只在需要更多信息时才下钻。

### 2. 执行优先

节点要求做什么就做什么，不要用"输出报告"替代实际执行。

- 节点有动词（删除、迁移、重构、实现）→ **必须执行**
- 只有明确标成"调研/分析/设计"的节点 → 允许只产出文字
- 审计/分析结果写进 Memory 结构化字段，不要生成 `.md` 文件

### 3. 渐进式是系统级约定

所有读接口默认轻量返回。这不是"可选优化"，而是系统设计的核心。AI 客户端必须按照渐进式模式使用工具。

---

## 渐进式读取决策树

```
┌─ 我只是要恢复上下文？
│  → task_tree_resume
│
├─ 我不知道该看哪个节点？
│  → task_tree_focus_nodes 或 task_tree_list_nodes_summary
│
├─ 我知道父节点，想看下一层？
│  → task_tree_list_nodes(parent_node_id=...)
│
├─ 我知道子树根，想看局部？
│  → task_tree_list_nodes(subtree_root_node_id=..., max_relative_depth=2)
│
├─ 我需要节点概要？
│  → task_tree_get_node_context(preset=summary)
│
├─ 我需要历史决策和记忆？
│  → task_tree_get_node_context(preset=memory)
│
├─ 我需要执行证据？
│  → task_tree_get_node_context(preset=work)
│  → 或 list_node_runs → get_run(include_logs=true)
│
├─ 我要搜索历史经验？
│  → task_tree_smart_search(q=...)
│
└─ 我需要任务级参考文件或架构决策？
   → task_tree_get_task_context
```

---

## 任务拆解最佳实践

### 树形优先，避免平铺

```
推荐：多层树形
"重构用户模块"
├─ "数据层"
│  ├─ "迁移 user 表结构"
│  ├─ "更新 repository 接口"
│  └─ "补充迁移测试"
├─ "API 层"
│  ├─ "重写 user CRUD 接口"
│  └─ "更新 OpenAPI spec"
└─ "前端适配"
   ├─ "更新用户列表页"
   └─ "更新用户详情页"

不推荐：平铺同级
"迁移 user 表" / "更新 repository" / "补充测试" / "重写接口" / ...
```

### 拆解原则

| 原则 | 说明 |
|------|------|
| 粒度控制 | 每个 leaf 节点 1-4 小时工作量 |
| 结构清晰 | group 节点组织结构，leaf 节点承载执行 |
| 依赖显式 | 有依赖关系时用 `depends_on_keys` |
| 阶段划分 | 大任务按阶段组织（数据层 → API层 → 前端） |

### 创建技巧

1. **一步到位**：优先 `task_tree_create_task(stages + nodes)` 一次创建完整骨架
2. **批量追加**：追加节点用 `task_tree_batch_create_nodes`，避免多次往返
3. **先 group 后 leaf**：先建组织结构，再往里补执行节点
4. **预演校验**：不确定时先 `dry_run=true`，确认结构正确再落库
5. **key 复用**：创建时设置 `key`，依赖通过 `depends_on_keys` 引用

### 示例：一次性创建

```json
{
  "title": "实现用户注册",
  "goal": "完成用户注册功能的全栈实现",
  "stages": [
    {"title": "后端", "key": "backend"},
    {"title": "前端", "key": "frontend"}
  ],
  "nodes": [
    {"title": "设计数据模型", "kind": "leaf", "stage_key": "backend", "key": "model"},
    {"title": "实现 API 接口", "kind": "leaf", "stage_key": "backend", "key": "api", "depends_on_keys": ["model"]},
    {"title": "编写单元测试", "kind": "leaf", "stage_key": "backend", "key": "test", "depends_on_keys": ["api"]},
    {"title": "实现注册表单", "kind": "leaf", "stage_key": "frontend", "key": "form", "depends_on_keys": ["api"]},
    {"title": "集成测试", "kind": "leaf", "stage_key": "frontend", "depends_on_keys": ["form", "test"]}
  ]
}
```

---

## 执行最佳实践

### 标准执行流程

```
1. task_tree_claim_and_start_run(node_id)         # 领取 + 开始
2. 实际执行操作                                     # 写代码、运行命令等
3. task_tree_progress(progress, log_content="...")  # 上报进度 + 自动记录执行过程
4. task_tree_complete(node_id, memory:{...})        # 完成 + 写入结构化 Memory
```

> 无需手写 execution_log，`progress(log_content)` 写入的 run_logs 会在 `preset=full` 时自动聚合。

### 执行规则

| 规则 | 说明 |
|------|------|
| 必须 claim | 先 `claim_and_start_run` 再执行，不要跳过 |
| 必须 complete | `progress(1.0)` 不等于完成，只有 `complete` 才标记 done |
| 不要空完成 | `summary_text` 不能只写"已完成"，要有实质内容 |
| 内联 Memory | `complete` 支持 `memory:{...}` 参数，可省掉单独的 patch 调用 |

### 自主推进

节点 complete 后：

1. 先看 `resume` 或当前 focus/children/subtree
2. 在依赖已满足的 `ready` 节点中选最合理的下一步
3. `recommended_action` 是建议而非强制指令
4. 有多个 ready 时，按结构顺序和真实依赖判断
5. 涉及不可逆操作或跨阶段切换时停下确认

---

## 性能与效率

### 读取效率

| 做法 | 效果 |
|------|------|
| 先 `resume` 再按需下钻 | 单次请求获取全局概览，避免多次往返 |
| 用 `focus_nodes` 而不是整树扫描 | 只返回可执行节点 + 祖先链 |
| 用 `preset=summary` 而不是 `full` | 减少 90%+ 响应体积 |
| 用 `parent_node_id` 限定范围 | 只读直接子节点 |
| 用 `max_relative_depth=2` | 控制子树下钻深度 |
| 用 `cursor` 分页 | 大列表分批加载 |

### 写入效率

| 做法 | 效果 |
|------|------|
| `claim_and_start_run` 合并调用 | 比 `claim` + `start_run` 少一次往返 |
| `batch_create_nodes` 批量创建 | 比逐个 `create_node` 减少 N-1 次往返 |
| `complete(memory:{...})` 内联写入 | 比 `patch_node_memory` + `complete` 少一次调用 |
| `batch_create_stages` 批量创建阶段 | 同上 |
| `progress(log_content=...)` 自动记录 | 替代手写 execution_log，节省 1000-3000 output tokens |

### 产物（Artifact）效率

| 场景 | 做法 | Token 消耗 |
|------|------|-----------|
| AI 生成代码/文档 | 写到文件 → `create_artifact(uri=路径)` 记录链接 | ~30 tokens |
| ❌ AI 生成内容后上传 | 生成 → base64 编码 → `upload_artifact(content_base64=...)` | 数千 tokens（双倍浪费） |
| 外部截图/附件 | `upload_artifact` 上传 | 合理使用 |

**原则：`upload_artifact` 仅用于非 AI 生成的外部文件。AI 产物一律写文件 + 记链接。**

### 搜索效率

| 场景 | 推荐方式 |
|------|---------|
| 搜索历史结论 | `smart_search(q=关键词)` — FTS5 全文索引 |
| 搜索特定任务内 | `smart_search(q=..., task_id=...)` — 限定范围 |
| 搜索特定类型 | `smart_search(q=..., scope=memory)` — 只搜 Memory |
| 浏览可执行项 | `work_items` — 直接返回可领取的工作项 |

---

## 并发与 Lease 机制

### 工作原理

- `claim` / `claim_and_start_run` 会为节点设置 lease
- lease 有过期时间，防止长期占用
- 同一时间只有一个 agent 能持有节点的 lease
- `release` 主动释放 lease
- `sweep_leases` 清理过期 lease

### 并发规则

| 场景 | 行为 |
|------|------|
| 节点已被 claim | 返回 409 冲突 |
| 节点已有 running run | 返回 409 冲突 |
| Lease 过期 | 节点可被重新 claim |
| 主动释放 | `release` 后节点立即可用 |

### 最佳做法

1. 执行前总是 `claim_and_start_run`
2. 完成后 `complete` 会自动释放 lease
3. 异常退出 → lease 过期后自动释放，或调 `sweep_leases`
4. 需要暂停 → `release` 释放 lease，后续可重新 claim
5. 长任务 → 定期 `progress` 保持活跃状态

---

## 阻塞与故障处理

### 标记阻塞

```
task_tree_transition_node(node_id, action=block, message="依赖的 API 接口尚未就绪")
→ task_tree_patch_node_memory(node_id, blockers=["等待 API 团队完成 /users 接口"])
→ task_tree_release(node_id)
```

### 解除阻塞

```
task_tree_transition_node(node_id, action=unblock)
```

### 节点状态流转

| 当前状态 | action | 目标状态 |
|---------|--------|---------|
| ready/running | `block` | blocked |
| ready/running | `pause` | paused |
| ready/running/paused/blocked | `cancel` | canceled |
| blocked | `unblock` | ready |
| paused | `reopen` | ready |
| canceled | `reopen` | ready |

### 故障恢复策略

| 场景 | 处理方式 |
|------|---------|
| 执行中途失败 | 写 Memory 记录进度 → `release` → 后续重新 `claim` 继续 |
| 发现需求变更 | 更新 `instruction` → `release` → 重新执行 |
| 节点不再需要 | `transition_node(action=cancel)` |
| Run 异常中断 | 系统会在 `finish_run` 时自动清理状态 |
| Lease 泄漏 | `sweep_leases` 清理所有过期 lease |
| 搜索索引异常 | `rebuild_index` 重建 FTS5 全文索引 |

---

## Memory 详细度要求

### execution_log：系统自动聚合，零 token 成本

`execution_log` **不再由 AI 手写**。系统自动从 run_logs 聚合生成——AI 通过 `progress(log_content=...)` 正常上报进度，系统在读取 `preset=full` 时自动格式化。

**Token 节省对比**：

| 方式 | AI output tokens | 信息质量 |
|------|-----------------|---------|
| ~~手写 execution_log~~（已废弃） | 1000-3000 | 依赖 AI 自律 |
| AI 只写结构化字段 | 100-300 | 聚焦决策 |
| 系统自动捕获 run_log | **0** | 客观完整 |

### 执行过程记录方式

每完成一步实质操作，调用 `progress` 并传 `log_content`：

```
task_tree_progress(node_id, progress=0.3, log_content="分析 store_memory.go，确认需要改造 6 个接入点")
task_tree_progress(node_id, progress=0.6, log_content="实现 aggregateNodeRunLogs，go build exit 0")
task_tree_progress(node_id, progress=0.9, log_content="完成单元测试，全部通过")
```

系统自动聚合为（`preset=full` 时返回）：
```
[2026-04-14 10:30] (progress) [30%] 分析 store_memory.go，确认需要改造 6 个接入点
[2026-04-14 10:35] (progress) [60%] 实现 aggregateNodeRunLogs，go build exit 0
[2026-04-14 10:38] (progress) [90%] 完成单元测试，全部通过
[2026-04-14 10:40] (complete) 完成
```

### 读取策略

- `preset=summary/work/memory` 都**不含** execution_log
- 只有 `preset=full` 返回自动聚合的 execution_log
- 正常工作流无需读取

### AI 必须写的字段

| 字段 | 要求 | 写入时机 |
|------|------|---------|
| `summary_text` | 做了什么 + 量化结果，不能只写"已完成" | 完成节点时 |
| `decisions` | 关键决策及理由（为什么选择方案 A 而非 B） | 做出决策时 |
| `evidence` | 文件路径、命令输出、验证结果 | 完成验证后 |
| `next_actions` | 下一步行动，帮助后续 agent 衔接 | 完成或中断时 |
| `blockers` | 当前阻塞项（配合 `transition_node(action=block)` 使用） | 遇到阻塞时 |
| `risks` | 已知风险和注意事项 | 发现风险时 |

### 合格示例

```json
{
  "summary_text": "删除 5 个废弃文件，更新 2 处引用，npm run build 通过",
  "decisions": ["保留兼容别名 3 个月——下游 v2 API 仍引用旧路径"],
  "evidence": ["frontend/src/main.js:L12 已更新", "npm run build exit 0"],
  "next_actions": ["继续处理详情页拆分（节点 nd_xxx）"],
  "risks": ["兼容别名到期后需再次清理"]
}
```

> `execution_log` 和 `append_execution_log` 参数已废弃，传入会被静默忽略。

---

## 知识复用

### 搜索优先

开始新工作前，先搜索：

```
task_tree_smart_search(q="用户注册")
```

常见复用场景：

| 场景 | 搜索方式 |
|------|---------|
| 查找历史结论 | `smart_search(q=关键词)` |
| 查找某任务内的经验 | `smart_search(q=..., task_id=...)` |
| 查找 Memory 中的决策 | `smart_search(q=..., scope=memory)` |
| 查看节点历史摘要 | `get_node_context(preset=memory)` |
| 查看执行证据 | `get_node_context(preset=work)` |

### 避免重复劳动

1. 搜索相关历史 Memory，了解之前的结论和决策
2. 检查相似节点是否已经处理过类似问题
3. 复用 `task_context` 中的架构决策和参考文件
4. 长期有效的经验写入 `decisions` 字段，方便后续搜索
