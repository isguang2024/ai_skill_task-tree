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
