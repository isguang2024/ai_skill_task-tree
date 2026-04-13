# Task Tree — 统一仓库

统一管理 Task Tree 的两个版本：v1（Core）和 v2。

## 仓库结构

```
task-tree/
├── v1/                  # Task Tree V1 (Core 版本)
│   ├── backend/         # Go 后端
│   ├── frontend/        # Vue 前端
│   ├── skill/           # 技能文档
│   ├── task-tree-dxt/   # DXT 代理
│   └── AGENTS.md        # 协作约束
│
├── v2/                  # Task Tree V2 (新版本)
│   ├── backend/         # Go 后端（改进版）
│   ├── frontend/        # Vue 3 前端
│   ├── docs/            # 使用文档
│   ├── skill/           # 技能文档
│   ├── task-tree-dxt/   # DXT 代理
│   └── AGENTS.md        # 协作约束
│
├── .gitignore           # Git 配置
└── README.md            # 本文件
```

## 版本说明

### V1（Core）
- 路径：`v1/`
- 端口：`8879`（后端 API/MCP）、`5173`（前端 dev）
- 现状：维护中，已支持基础功能

### V2（新版）
- 路径：`v2/`
- 端口：`8880`（后端 API/MCP）、`5174`（前端 dev）
- 现状：主力版本，功能完善

## 快速启动

### V1
```powershell
cd v1/backend
$env:TTS_ADDR="127.0.0.1:8879"
go run ./cmd/task-tree-service serve

# 另一个终端
cd v1/frontend
npm run dev
```

### V2
```powershell
cd v2/backend
$env:TTS_ADDR="127.0.0.1:8880"
go run ./cmd/task-tree-service serve

# 另一个终端
cd v2/frontend
npm run dev
```

## 技能文件位置

- 全局 Claude Code：`C:\Users\Administrator\.claude\skills\task-tree\SKILL.md`
- 全局 Codex：`C:\Users\Administrator\.codex\skills\task-tree\SKILL.md`
- V1 本地：`v1/skill/SKILL.md`
- V2 本地：`v2/skill/SKILL.md`

## 协作约束

- 修改技能或新增 API/MCP 时，必须同步更新全局技能文件
- 除 `SKILL.md` 外，`v2/skill/docs/` 下文档（`task-tree-api.md`、`task-tree-best-practices.md`、`task-tree-tools.md`）也必须同步到全局技能目录
- 每个版本的 AGENTS.md 中有详细的协作原则
- 两个版本共享同一个远程仓库

## 参考文档

- V1 说明：见 `v1/AGENTS.md`
- V2 说明：见 `v2/AGENTS.md`
- V2 技能指南：`v2/skill/SKILL.md`
- V2 本地启动：`v2/docs/本地启动与使用.md`
