# skill

这个目录存放 Task Tree V2 的服务使用指南和参考文档。

## 文件说明

- [SKILL.md](SKILL.md)
  核心技能文档（≤200行），涵盖连接信息、工作流、行为规则和 Memory 约定。
- [docs/task-tree-tools.md](docs/task-tree-tools.md)
  完整 MCP 工具清单（含废弃工具）。
- [docs/task-tree-api.md](docs/task-tree-api.md)
  完整 HTTP API 参考。
- [docs/task-tree-best-practices.md](docs/task-tree-best-practices.md)
  最佳实践、决策树、Memory 示例。

## 使用方式

AI 每次加载只读取 `SKILL.md`，需要工具或 API 细节时按需 `Read` 对应的 `docs/` 文档。
