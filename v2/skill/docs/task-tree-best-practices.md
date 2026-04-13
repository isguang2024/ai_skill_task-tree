# Task Tree V2 — 最佳实践与决策树

> 本文件是按需查阅的参考文档，不在每次对话中加载。需要时用 `Read` 工具读取。

## 任务拆解最佳实践

### 树形优先，避免平铺

创建任务时，应该把工作拆成**多层树形结构**，而不是一堆同级节点：

```
✅ 推荐：树形结构（用 group + leaf）
"重构用户模块"（group）
├─ "数据层"（group）
│  ├─ "迁移 user 表结构"（leaf）
│  ├─ "更新 repository 接口"（leaf, depends_on: 上一个）
│  └─ "补充迁移测试"（leaf, depends_on: 上一个）
├─ "API 层"（group, depends_on: 数据层）
│  ├─ "重写 user CRUD 接口"（leaf）
│  └─ "更新 OpenAPI spec"（leaf）
└─ "前端适配"（group, depends_on: API 层）
   ├─ "更新用户列表页"（leaf）
   └─ "更新用户详情页"（leaf）
```

```
❌ 不推荐：全部平铺在同一级
"迁移 user 表" / "更新 repository" / "补充测试" / "重写接口" / "更新 spec" / ...
→ 看不出结构，不知道哪些有依赖，进度聚合无意义
```

### 创建节点的技巧

1. **一次创建多个**：用 `batch_create_nodes` 而不是多次 `create_node`
2. **先创建 group，再往里填 leaf**：指定 `parent_node_id` 构建层级
3. **声明依赖**：优先用 `depends_on_keys`（创建期无需先拿 node_id），也可用 `depends_on: [node_id]`
4. **阶段可选**：简单任务不需要 Stage，直接用节点树就够了
5. **create_task 可一步到位**：`stages` + `nodes` 都可以在创建任务时传入

### 节点粒度

| 粒度 | 判断 |
|------|------|
| 太粗 | "实现用户模块" — 包含多个独立工作，应拆分 |
| 合适 | "迁移 user 表结构" — 1-4 小时，单一目标，可验收 |
| 太细 | "在第 42 行加一个字段" — 应该合并到更大的节点中 |

### 何时追加子节点

- 执行中发现工作量超预期 → 把剩余部分拆成子节点
- 发现预料之外的依赖工作 → 追加为新子节点
- 用户调整方向或新增需求 → 追加子节点而不是修改已完成的节点

## 执行优先原则

**节点要求你做什么，你就去做什么。不要用"输出报告"替代"实际执行"。**

- 节点描述中有动词（删除/迁移/重构/实现/修改）→ **必须执行对应操作**
- 只有明确标注"调研/分析/设计"的节点 → 才可以仅输出文档
- 审计/分析类结果直接写进 Memory 的结构化字段，不要每个节点都生成 `.md` 报告

## 自主推进规则

节点 complete 后判断是否继续：

```
1. 当前阶段还有 ready 节点？
   ├─ 有 → 自动 claim 下一个，继续执行
   └─ 没有 → 当前阶段全部完成
2. 下一个节点有未完成的前置依赖？
   ├─ 有 → 跳过，找其他可执行节点
   └─ 无 → 继续
3. 涉及不可逆操作（删除/迁移）？
   ├─ 是 → 暂停确认
   └─ 否 → 直接执行
4. 阶段全部完成？
   └─ 向用户报告，询问是否进入下一阶段
```

**不需要确认**：同阶段连续节点、纯代码修改/构建/测试
**必须确认**：跨阶段切换、不可逆操作、方向可能有误

## 检索与知识复用

### 开始工作前先搜索

在创建新任务或执行新节点前，用 `smart_search` 检查是否有相关历史：

```
smart_search(q: "用户登录")
→ 可能找到之前做过的登录相关节点，其中包含：
  - decisions: "选了 JWT 不选 Session，因为需要支持多端"
  - risks: "token 过期时间设太长有安全风险"
  - evidence: "前端用 pinia 存 token，见 stores/auth.ts"
```

这些信息可以避免重复劳动和重复踩坑。

### 检索场景速查

| 场景 | 工具 | 说明 |
|------|------|------|
| 搜索关键词 | `smart_search(q: "xxx")` | FTS5 全文检索，覆盖标题/指令/Memory/日志 |
| 查看历史决策 | `get_node_context(node_id)` | Memory.decisions 字段 |
| 查看执行细节 | `get_node_context(node_id)` | Memory.execution_log 字段 |
| 查看原始日志 | `list_node_runs` → `get_run` | 最细粒度的运行记录 |
| 查看阶段全貌 | `get_node_context(stage_node_id)` | 阶段聚合所有子节点 Memory |
| 查看项目概览 | `project_overview(project_id)` | 项目下所有任务统计 |
| 查看剩余工作 | `get_remaining(task_id)` | 按状态统计节点数 |

## Memory 详细度要求

**合格的 Memory 示例：**
```json
{
  "summary_text": "删除 Instructions/ 目录（5个文件）和 iconify-loader.ts，pnpm build + vue-tsc 通过",
  "execution_log": "1. grep 搜索引用 → 0处\n2. rm -rf Instructions/\n3. 删除 iconify-loader.ts\n4. pnpm build 通过",
  "conclusions": ["Instructions/ 全目录无引用，属于历史遗留"],
  "decisions": ["直接 rm -rf 而非逐个删除"],
  "evidence": ["git status: 6 files deleted", "pnpm build exit 0"],
  "next_actions": ["继续节点 1.3 清理重复功能"]
}
```

**不合格**：`{ "summary_text": "已完成", "evidence": ["已完成"] }`

各字段要求：
- **summary_text**：做了什么 + 量化结果，不要只写"已完成"
- **execution_log**：详细过程，改了哪些文件、执行了什么命令、遇到的问题和解决方案
- **conclusions**：分析结论和判断依据
- **decisions**：关键决策和选择原因（"选了 A 不选 B，因为…"）
- **evidence**：文件路径、命令输出、验证结果
- **risks / blockers**：诚实记录，不要省略

## 典型决策树

```
1. 任务已存在?
   ├─ 是 → resume(task_id) → 看 recommended_action
   └─ 否 → create_task(stages + nodes)
        └─ 用 batch_create_nodes 创建树形结构

2. 获得下一节点 → claim_and_start_run
   └─ 开始前 smart_search 检查是否有相关历史

3. 执行节点要求的实际操作

4. 中间进度 → progress(log_content: "详细日志")

5. 完成 → complete(memory: { summary_text, execution_log, ... })
   └─ 响应中 .next 自动包含下一节点

6. 继续? → 回到步骤 2（从 .next 中获取 node_id）
```

## 完成节点的正确方式

**错误**：只调 `progress(1.0)` — 状态仍是 running
**正确**：调 `complete(message, memory)` — 明确标记 done，生成事件，解锁下一节点

## checkpoint 门禁节点

当节点是 `kind=leaf + role=checkpoint` 且配置了 `metadata.checkpoint_spec.required_commands`：

- `complete` 必须提供 `result_payload.commands_verified`
- 且 `commands_verified` 必须覆盖 `required_commands`
- 否则服务端会返回 `409`，节点不会完成

## 遇到阻塞

```
1. transition_node(node_id, action: "block", message: "原因")
2. Memory 中记录 blockers
3. release(node_id)
```

解除：`transition_node(node_id, action: "unblock")`

## 常见反模式

| 反模式 | 正确做法 |
|--------|---------|
| 所有节点平铺在同一级 | 用 group/leaf 构建树形层级 |
| 每个节点输出一个 `.md` 报告 | 结果写进 Memory 结构化字段 |
| 不写 execution_log | 详细记录执行过程，供下次 resume 回读 |
| 跳过 claim 直接执行 | 先 claim_and_start_run 再执行 |
| 用 progress(1.0) 代替 complete | 必须调 complete 才标记完成 |
| 不搜索就开始新工作 | 先 smart_search 检查历史 |
| Memory 只写"已完成" | 写清做了什么、量化结果、关键决策 |
| 把大任务塞进一个节点 | 拆成 1-4 小时的子节点 |
