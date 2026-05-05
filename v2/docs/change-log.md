## 2026-04-12 执行层与 UI 契约推进

## 2026-04-12 V2 技能补充 Memory 约定

### 本次改动
- 在 `skill/SKILL.md` 中新增 Memory 维护约定，明确 V2 的 `manual_note_text` 属于人工备注，而 `summary / decisions / risks / next_actions` 应优先由 AI 或系统生成。
- 规定了 AI 的默认行为：节点完成后刷新节点 Memory，阶段推进后刷新阶段 Memory，任务收尾或方向变化后刷新任务 Memory。
- 在 `docs/技能与MCP说明.md` 里同步补充这条使用约定，避免后续技能用户把 Memory 误解为纯人工输入区。

### 下次方向
- 如果后续给 Memory 补专门的“生成/刷新”接口或 MCP 工具，应把这些触发点从文档约定升级为可执行能力。
- UI 文案后面也应继续区分“人工备注”和“AI Memory”，减少使用歧义。

## 2026-04-12 V2 文档体系重写

### 本次改动
- 重写 `AGENTS.md`，把仓库说明完全切到 `task-tree-v2`，统一端口为 `8880 / 5174`，并明确 HTTP、MCP、DXT、Skill 的边界。
- 新增 `docs/README.md`、`docs/本地启动与使用.md`、`docs/技能与MCP说明.md`，分别覆盖项目总览、本地启动/发布方式、技能用户与 MCP 用户的接入说明。
- 重写 `backend/docs/http-mcp-parity.md`，明确区分“已 HTTP+MCP 对齐”的核心能力与当前仍是 `HTTP only` 的 Run / Memory / context 能力。
- 补全 `backend/docs/mcp-open-manifest.txt`，使其与当前实际 MCP 暴露工具一致，包含阶段、重排、移动等 V2 工具。

### 下次方向
- 如果未来给 Run / Memory 补 MCP 工具，优先同步更新 `http-mcp-parity.md`、`mcp-open-manifest.txt` 和 `skill/SKILL.md`。
- 如果还要支持对外发布，可继续补一份更偏运维的部署文档，覆盖构建产物、日志和数据目录备份。

### 本次改动
- 补齐执行层基础设施：新增 `008_run_tables.sql`，实现 `store_runs.go`，打通 `startRun`、`finishRun`、`addRunLog`、`getRun`、`listNodeRuns`。
- 新增执行层 HTTP API：`POST/GET /v1/nodes/{nodeId}/runs`、`GET /v1/runs/{runId}`、`POST /v1/runs/{runId}/finish`、`POST /v1/runs/{runId}/logs`。
- 旧 `progress/complete` 接口开始自动生成 synthetic run，避免历史调用路径没有执行记录。
- 修复 UI smoke test 契约：旧表单页补上统一 `id="app"` 壳层，`go test ./...` 已恢复全绿。

### 下次方向
- 给 MCP 层补 run 相关工具，避免 HTTP 与 MCP 能力继续分叉。
- 继续推进 `Phase 4：记忆层`，把 run 结果与节点级记忆/摘要关联起来。
- 视需要把 `block/cancel/reopen` 也映射到 run 生命周期，补齐执行层闭环。

## 2026-04-12 前端 V2 读模型接入

### 本次改动
- 前端 API 层切到 V2：`src/api.js` 增加 `resume/context/overview/stages/runs/memory` 包装函数，并统一节点规范化与 SSE dirty envelope 解析。
- `TaskDetail.vue` 默认改为消费 V2 `resume` 聚焦树，页头直接展示 task memory、current stage memory，节点详情改为按需拉取 `/v1/nodes/{id}/context`。
- `TaskDetail.vue` 的 SSE 刷新策略从“事件后全量重拉”改成优先按 dirty 区域局部刷新，只有命中 `resume/task/runs/artifacts` 时才重载整页。
- `TaskList.vue` 改为优先使用 `/v1/projects/{id}/overview`，任务卡片开始展示当前阶段与 task memory 摘要。

### 验证
- `go test ./...` 在 `backend` 下通过。
- `npm install && npm run build` 在 `frontend` 下通过。

### 下次方向
- 继续做 `Phase 6` 剩余前端节点：阶段管理、Run 管理、Memory 展示编辑。
- 处理前端构建产物体积偏大问题，后续考虑按路由拆包。

## 2026-04-12 前端 V2 阶段与运行面板

### 本次改动
- 在 `TaskDetail.vue` 中新增阶段管理卡片，支持列出阶段、创建阶段、激活阶段，并修正前端 `activateStage` 实际路径为 `/tasks/{taskId}/stages/{stageNodeId}/activate`。
- 在节点详情操作区接入 Run 面板，支持开始 Run、结束 Run、追加日志，并联动刷新节点 context 与任务 resume。
- 任务详情页顶部新增任务最近运行概览，阶段和执行层的 V2 数据已经能在同一页面闭环操作。

### 验证
- `npm run build` 通过。
- `go test ./...` 通过。

### 下次方向
- 继续补 `7/6`：记忆层 UI 展示与编辑。
- 视页面复杂度决定是否把阶段 / Run / Memory 拆成子组件，降低 `TaskDetail.vue` 体积。

## 2026-04-12 前端 V2 记忆层接入

### 本次改动
- 在 `TaskDetail.vue` 中新增 task/stage/node 三层 Memory 卡片，直接展示当前 summary，并补充 decisions / risks / next actions 的轻量标签。
- 接入 `PATCH /tasks/{id}/memory`、`PATCH /stages/{id}/memory`、`PATCH /nodes/{id}/memory`，支持在页面内编辑 `manual_note_text`。
- 节点 context、当前阶段 memory、任务 memory 的表单值开始统一同步，阶段与运行动作后的刷新也能带回最新 memory。

### 验证
- `npm run build` 通过。
- `go test ./...` 通过。

### 下次方向
- 如果继续推进，会把 `7/7` 的 SSE 局部刷新收得更细，减少不必要的 resume 重拉。
- 然后直接进入 `Phase 7` 的测试与验证节点。 

## 2026-04-12 Phase 7 测试收尾

### 本次改动
- 在 `service_test.go` 新增基础验证 `TestMigrationDuplicateVersionAndExpectedVersionConflict`，补上 `expected_version` 冲突和重复 migration version 的显式回归测试。
- 复核并沿用现有 `TestStagesFlow`、`TestRunRoutesAndSyntheticCompatibility`、`TestMemoryAndReadModelFlow`，覆盖阶段层、执行层、记忆层、读取层的 V2 核心路径。
- 前端继续通过 `npm run build` 校验，确保 Phase 6 新接的 V2 页面能力没有因为测试层改动产生构建回归。

### 验证
- `go test ./...` 通过。
- `npm run build` 通过。

### 下次方向
- 当前只剩性能/工程化类非阻塞事项，例如前端 bundle 体积偏大、`TaskDetail.vue` 可继续拆组件。
- 如果还要继续推进，我会优先处理前端拆包和组件分层。 

## 2026-04-14 渐进读取全量收口

### 本次改动
- 把节点/运行/产物读取统一成默认轻量模型：`/tasks/{id}/nodes` 默认返回 summary，补齐 `parent_node_id`、`subtree_root_node_id`、`max_relative_depth`，并把 run/artifact 路由都改成支持 `view_mode`、`limit`、`cursor`。
- 节点上下文新增 `summary/work/memory/full` 预设，前端节点选中默认只拿 `preset=work`，任务详情 full 模式直接复用 `resume.full_tree`，项目列表改走 `summary_with_stats`，去掉项目页 fan-out 概览请求。
- 内置 AI 和 MCP 一并改成摘要优先：新增 `resume_task`、`list_nodes_summary`、`get_node_context`、`claim_and_start_run`、`batch_create_nodes`、`import_plan`、`smart_search` 等工具入口，并补上子树展开、run logs 开关、artifact summary 的参数模型。

### 验证
- `go test ./internal/tasktree -run 'TestDependsOnSummaryAndExecutionOrder|TestFocusFilterAppliesBeforePagination|TestResumeHonorsTreePagingAndResumeContextSiblings|TestNodesDefaultSummaryAndSubtreeFilters|TestNodeContextPresetRunAndArtifactSummary|TestProjectsSummaryWithStats|TestSmokeFlow|TestRunRoutesAndSyntheticCompatibility'` 通过。

### 下次方向
- 继续把前端节点详情拆成按 tab 懒加载，避免 `TaskDetail.vue` 在选中节点时仍然预取过多上下文。
- 如果要继续推进 HTTP/MCP 对齐，可以补 `list_children` / `list_subtree_summary` 这类显式子树接口，减少调用方手拼过滤参数。 

## 2026-04-14 Resume 轻量默认与节点页懒加载

### 本次改动
- `resume` 新增 `include` 模型，默认不再附带 `events`、`runs`、`artifacts`、`next_node_context` 这类重内容；HTTP 路由、MCP 和内部调用统一走同一套解析与执行逻辑。
- 前端 `TaskDetail.vue` 改成按当前 tab 取数：节点页按 `preset=work`，Memory 页按 `preset=memory`，事件页和产物页再分别补拉 `/events`、`/tasks/{id}/artifacts`，同时 `resume` 只在需要的 tab 上显式请求对应 include。
- 新增 `/events` 与 task artifacts 的前端 API wrapper，并补测试锁住 `resume` 的轻量默认与 `include` 按需展开行为。

### 验证
- `go test ./internal/tasktree -run 'TestDependsOnSummaryAndExecutionOrder|TestFocusFilterAppliesBeforePagination|TestResumeHonorsTreePagingAndResumeContextSiblings|TestResumeIncludesOptInHeavySections|TestNodesDefaultSummaryAndSubtreeFilters|TestNodeContextPresetRunAndArtifactSummary|TestProjectsSummaryWithStats|TestSmokeFlow|TestRunRoutesAndSyntheticCompatibility'` 通过。
- `npm run build` 通过。

### 下次方向
- 继续把节点页里仍然放在 `work` preset 里的 runs/event/artifact 细项拆得更细，避免“节点 tab”本身继续捎带不必要数据。
- 如果继续做协议层收口，下一步适合补 `task_tree_list_children` / `task_tree_list_subtree_summary` 这类更显式的 MCP 工具。 

## 2026-04-14 剩余读模型修复收口

### 本次改动
- 修正 `/resume` 的过滤门控，把 `parent_node_id`、`subtree_root_node_id`、`max_relative_depth`、`has_children` 也纳入过滤树判定，避免“参数能传但实际不生效”。
- `resume` 的 `current_stage_memory` 改成和 `task_memory` 一样的轻摘要结构，并补上 `summary` 别名；`projectOverview` 去掉逐任务 `getRemaining/findNode/getTaskMemory` 的 N+1，改成节点汇总、阶段摘要、memory 摘要的批量查询。
- 前端修正 `nextNode` 在无后继节点时的陈旧值问题，把 `listAllNodes` 收口成 `listNodeDetails`（保留兼容别名），并把内置 AI 默认读法从 `preset=work` 收紧到先 `preset=summary`。

### 验证
- `go test ./internal/tasktree -run 'TestDependsOnSummaryAndExecutionOrder|TestFocusFilterAppliesBeforePagination|TestResumeHonorsTreePagingAndResumeContextSiblings|TestResumeIncludesOptInHeavySections|TestNodesDefaultSummaryAndSubtreeFilters|TestNodeContextPresetRunAndArtifactSummary|TestProjectsSummaryWithStats|TestSmokeFlow|TestRunRoutesAndSyntheticCompatibility|TestMemoryAndReadModelFlow'` 通过。
- `npm run build` 通过。

### 下次方向
- 继续把 `projectOverview` 的 task list 也裁成更明确的 summary DTO，避免列表页继续持有不必要字段。
- 如果要继续压前端体积，下一步直接做 `TaskDetail.vue` 的拆组件和按路由/按 tab 分包。 

## 2026-04-18 Codex 用量记录补充

### 本次改动
- 给 `v2/skill/SKILL.md` 增加了 Codex 用量规则，要求每个节点完成后记录用量，优先写节点增量，没有节点级原生值时退回线程累计值。
- 新增 `v2/skill/docs/codex-usage.md`，把 `~/.codex/state_5.sqlite` 和 `CODEX_THREAD_ID` 的用量获取步骤单独拆出来。
- 在读取路径索引里补了 `codex-usage` 路由，方便按需加载。

### 下次方向
- 现在能稳定拿到的是线程累计用量，后续如果 Codex 暴露节点级字段，再把文档升级成原生节点账单口径。
- 如果要继续收口，可以把用量步骤再细拆成“查询脚本”和“节点 Memory 写法”两个独立小文件。

## 2026-04-18 技能文档拆分收口

### 本次改动
- 将 `v2/skill/SKILL.md` 压缩成入口页，只保留必守规则、连接信息和按需读取路由，减少默认加载体积。
- 把原本堆在大文件里的内容拆成更细的子文档：读取路径、任务拆解、执行、Memory、阻塞恢复、扩树、陷阱，以及 MCP / HTTP 的分域索引。
- 修正了文档里的旧字段名和旧约定，把 `create_project`、`create_stage`、`move_node`、`create_artifact`、`get_task` 等说明对齐到当前工具签名。

### 下次方向
- 如果还要继续压上下文，可以把 `task-tree-tools-read.md` 和 `task-tree-api-task.md` 再拆一层，按“恢复 / 导航 / 列表 / 详情”继续细分。
- 目前入口已经收口，后续更值得补的是示例覆盖和实现一致性的持续校对。

## 2026-04-18 用量字段与接口打通

### 本次改动
- 后端新增 `usage_tokens` 字段，覆盖 `tasks` / `nodes` / `node_runs`，并在 `finish_run` 与节点完成链路里写回，任务总用量按节点树自底向上汇总。
- HTTP 和 MCP 完成接口补齐 `usage_tokens` 入参，`get_task` / `project_overview` / `list_nodes` / `list_runs` 等返回体也同步带上用量字段。
- 任务列表、任务详情、节点树和搜索结果的 UI 读模型都接入了 `usage_tokens`，列表页和详情页可以直接看到总用量。
- 补了 MCP/HTTP 回归测试，覆盖 `finish_run`、`complete`、`get_task`、节点 run 列表等用量回传与聚合结果。

### 下次方向
- 如果后面要做“自动采集 Codex 节点用量”，可以再加启动时快照和结束时差值计算，当前版本先以显式传入/接口回传为准。
- 现有历史任务没有原始 token 来源，旧数据默认还是 0，只有这次改动之后的新运行会稳定记录。

## 2026-04-18 Codex 用量自动采集

### 本次改动
- 新增 Codex 用量采集逻辑，运行开始时读取 `CODEX_THREAD_ID` 对应线程的累计用量快照，结束时按差值自动回填 `usage_tokens`。
- 默认从 `CODEX_STATE_DB_PATH` 读取本地 sqlite；未配置时回退到 `CODEX_HOME/.codex/state_5.sqlite`，保证本地 Codex 环境可以直接工作。
- `finish_run`、节点完成接口、MCP 完成工具都保留显式 `usage_tokens` 覆盖能力，自动采集只作为默认值，不会挡住手工回填。
- 新增测试覆盖自动快照、差值计算和任务树汇总，避免后续回归。

### 下次方向
- 目前能稳定拿到的是线程累计用量，后续如果 Codex 暴露节点级原生 token 字段，再把自动采集切换成原生节点账单口径。
- 如果还要继续收口，可以把“读取 sqlite”和“写入 Memory 模板”再拆成两个更小的说明文件，进一步降低按需加载成本。

## 2026-04-18 Codex 用量流程分流

### 本次改动
- 把 Codex 用量说明拆成两条路由：`task-tree` 内部节点差值流程，和其他代理独立自取流程。
- 新增 `codex-usage-self-fetch.md`，专门描述不依赖 task-tree 生命周期、仅凭 `CODEX_THREAD_ID` 和本机 sqlite 自取 `tokens_used` 的方法。
- 在入口技能和读取路径索引里同时标明两种读法，避免后续代理把“线程差值”误当成唯一流程。

### 下次方向
- 如果后面要给别的代理继续降上下文，可以把“自取流程”再拆成“环境定位”“sqlite 查询”“节点差值”三个更小文件。
- 目前文档已经分流，后续更重要的是把实现侧的自动采集和文档口径保持同步。

## 2026-04-18 技能能力说明补充

### 本次改动
- 在 `v2/skill/README.md` 里补了技能能力说明，明确写出 Task Tree 同时支持 MCP 和 HTTP 两种入口。
- 说明里补充了两种入口的职责边界：MCP 负责节点、任务、run、Memory 的读写和执行，HTTP 负责同一套数据查询、提交和管理。

### 下次方向
- 如果还要继续压上下文，可以把 README 再缩成更纯的索引页，把能力说明挪到单独的 `docs/skill-capabilities.md`。
- 后续新能力如果同时出现在 MCP 和 HTTP，要同步更新这份说明，避免入口文档和实现分叉。

## 2026-04-24 顶部导航与主题色调整

### 本次改动
- 将前端顶部栏改成固定单行布局，左侧显示站点图标和名称，中间保留面包屑，右侧保留操作按钮，避免侧栏模式下顶部内容被挤成多行。
- 去掉了侧栏顶部的站点标题占位，减少和顶部品牌区的重复展示。
- 通过 `themeOverrides` 把主色显式拉回默认蓝系，避免按钮和高亮继续沿用偏黄的视觉。

### 下次方向
- 如果后续还要进一步对齐截图效果，可以继续把顶部面包屑和右侧操作区做更紧凑的响应式收缩。
- 当前只改了 `App.vue`，如果其他页面里还有局部暖色高亮，再继续逐页统一主题变量。

## 2026-04-14 渐进式读取第二阶段收口

### 本次改动
- `/resume` 继续收轻：默认只保留 `task`、`task_memory_summary`、`current_stage`、`tree`、`remaining`、`recommended_action`、`next_node_summary`，`task_memory` / `stage_memory` / `events` / `runs` / `artifacts` / `next_node_context` 全部改成显式 `include`。
- MCP 补齐显式局部展开工具：新增 `task_tree_list_children` 与 `task_tree_list_subtree_summary`，`task_tree_resume` 也同步支持 `parent_node_id`、`subtree_root_node_id`、`max_relative_depth`、`has_children` 和新的 memory include 项。
- 内置 AI 与前端节点页继续拆轻：AI `get_task` 默认不再隐式拼树；`TaskDetail.vue` 的 node tab 改成 `preset=summary + 单独 runs`，run 日志改成按需加载；项目概览任务列表也裁成 summary DTO，前端只消费 `remaining/current_stage/memory.summary` 这类摘要字段。

### 验证
- `go test ./internal/tasktree -run 'TestMemoryAndReadModelFlow|TestResumeIncludesOptInHeavySections|TestMCPChildrenAndSubtreeSummaryTools|TestAIToolGetTaskRequiresExplicitTree|TestDependsOnSummaryAndExecutionOrder|TestFocusFilterAppliesBeforePagination|TestResumeHonorsTreePagingAndResumeContextSiblings|TestNodesDefaultSummaryAndSubtreeFilters|TestNodeContextPresetRunAndArtifactSummary|TestProjectsSummaryWithStats|TestSmokeFlow|TestRunRoutesAndSyntheticCompatibility'` 通过。

### 下次方向
- `TaskDetail.vue` 仍然偏大，后续可以把节点详情、memory、run 详情拆成独立组件，继续降低单文件复杂度。
- 如果继续做性能收尾，下一步优先看前端大 chunk 分包和项目页定时刷新策略。 

## 2026-04-14 前端分包与详情页拆分收尾

### 本次改动
- 路由组件全部改成按页懒加载，并在 `vite.config.js` 里补上 `vue-core / vue-router / naive-ui` 的 manual chunk，构建产物从单包切成了多页面 chunk。
- `TaskDetail.vue` 把 `events / memory / artifacts` 三个 tab 抽成独立组件，节点页继续保留主流程逻辑，run 日志仍是按需加载，文件复杂度继续下降。
- `TaskDetail.vue` 的 node 主面板也已接到独立的 `TaskNodeTab.vue`，详情页主文件现在只负责状态编排和接口调度，后续拆 run 详情或节点概览都可以继续在子组件内推进。
- `TaskList.vue` 继续沿用 summary DTO 读法，项目页与详情页的前端入口都保持在轻量读取模型上。

### 验证
- `npm run build` 通过。

### 下次方向
- 当前最大的剩余 chunk 还是 `naive-ui`，如果还要继续压体积，下一步应该评估按需引入或更细的 vendor 拆分。
- `TaskDetail.vue` 的主文件已经缩到状态层，下一步更适合继续拆 `TaskNodeTab.vue` 里的 run 详情或节点概览子区块。 

## 2026-04-14 文档与技能同步收尾

### 本次改动
- 把渐进式读取第二阶段收口后的最新规则写回仓库文档，统一了 `/resume` 轻默认、`children / subtree` 下钻、`get_task(include_tree)`、`get_run(include_logs)`、`preset=summary/memory/work/full` 等最新约定。
- 更新了技能主文档和 `skill/docs` 三份参考文档，把 MCP 清单、推荐读取顺序、行为规则、HTTP 边界统一成和当前实现一致的版本。
- 将仓库内最新 `skill/` 文档同步到 `C:\\Users\\Administrator\\.codex\\skills\\task-tree\\` 与 `C:\\Users\\Administrator\\.claude\\skills\\task-tree\\`，避免全局技能目录继续停留在旧规则。

### 下次方向
- 如果继续做发布侧收尾，优先检查 DXT 包内引用的技能/说明是否也需要跟着同步版本说明。
- 工程侧剩余工作主要还是前端 chunk 继续优化和 `TaskNodeTab.vue` 内部再细拆。 
