# Task Tree V2 — 完整 HTTP API 参考

> 本文件是按需查阅的参考文档，不在每次对话中加载。需要时用 `Read` 工具读取。

## 项目

```
GET    /v1/projects                       — 列出项目
POST   /v1/projects                       — 创建项目
GET    /v1/projects/{id}                  — 项目详情
PATCH  /v1/projects/{id}                  — 更新项目
DELETE /v1/projects/{id}                  — 删除项目
GET    /v1/projects/{id}/overview         — 项目概览
GET    /v1/projects/{id}/tasks            — 项目下的任务列表
```

## 任务

```
GET    /v1/tasks                          — 列出任务
POST   /v1/tasks                          — 创建任务（支持 stages + nodes 一步到位）
GET    /v1/tasks/{id}                     — 任务详情
PATCH  /v1/tasks/{id}                     — 更新任务
DELETE /v1/tasks/{id}                     — 软删除
POST   /v1/tasks/{id}/hard                — 硬删除
POST   /v1/tasks/{id}/restore             — 恢复
POST   /v1/tasks/{id}/transition          — 状态流转
GET    /v1/tasks/{id}/resume              — 任务上下文（核心）
GET    /v1/tasks/{id}/remaining           — 剩余进度
GET    /v1/tasks/{id}/next-node           — 下一个可执行节点
GET    /v1/tasks/{id}/resume-context      — 节点级恢复上下文（祖先链 + Memory + 最近运行）
GET    /v1/tasks/{id}/wrapup             — 获取任务收尾总结
POST   /v1/tasks/{id}/wrapup             — 写入/更新任务收尾总结
GET    /v1/tasks/{id}/events/stream       — SSE 事件流
```

## 任务 Memory

```
GET    /v1/tasks/{id}/memory              — 获取
PATCH  /v1/tasks/{id}/memory              — 更新
POST   /v1/tasks/{id}/memory/snapshot     — 快照
```

## 阶段

```
GET    /v1/tasks/{id}/stages              — 列出阶段
POST   /v1/tasks/{id}/stages              — 创建阶段
POST   /v1/tasks/{id}/stages/{sid}/activate — 激活阶段
GET    /v1/stages/{id}/memory             — 阶段 Memory
PATCH  /v1/stages/{id}/memory             — 更新阶段 Memory
POST   /v1/stages/{id}/memory/snapshot    — 快照阶段 Memory
```

## 节点

```
GET    /v1/tasks/{id}/nodes               — 列出节点（view_mode: slim/summary/detail/events）
POST   /v1/tasks/{id}/nodes               — 创建节点
POST   /v1/tasks/{id}/nodes/batch         — 批量创建节点
POST   /v1/tasks/{id}/reorder             — 重排序
GET    /v1/nodes/{id}                     — 节点详情
PATCH  /v1/nodes/{id}                     — 更新节点
GET    /v1/nodes/{id}/context             — 节点上下文（核心）
POST   /v1/nodes/{id}/transition          — 状态流转（block/pause/reopen/cancel/unblock）
POST   /v1/nodes/{id}/move                — 移动节点
POST   /v1/nodes/{id}/progress            — 上报进度
POST   /v1/nodes/{id}/complete            — 完成节点（支持内联 memory + 自动 next_node）
POST   /v1/nodes/{id}/block               — 标记阻塞
POST   /v1/nodes/{id}/claim-and-start-run — 领取+开始运行
POST   /v1/nodes/{id}/claim               — 领取
POST   /v1/nodes/{id}/release             — 释放
POST   /v1/nodes/{id}/retype              — 类型转换
```

## 节点 Memory

```
GET    /v1/nodes/{id}/memory              — 获取
PATCH  /v1/nodes/{id}/memory              — 更新
POST   /v1/nodes/{id}/memory/snapshot     — 快照
```

## 运行

```
POST   /v1/nodes/{id}/runs                — 启动运行
GET    /v1/nodes/{id}/runs                — 列出运行
GET    /v1/runs/{id}                      — 运行详情
POST   /v1/runs/{id}/finish               — 结束运行
POST   /v1/runs/{id}/logs                 — 追加日志
```

## 产物

```
GET    /v1/tasks/{id}/artifacts           — 任务产物
GET    /v1/nodes/{id}/artifacts           — 节点产物
POST   /v1/tasks/{id}/artifacts           — 创建产物
POST   /v1/tasks/{id}/artifacts/upload    — 上传产物
GET    /v1/artifacts/{id}/download        — 下载产物
```

## 全局

```
GET    /v1/work-items                     — 待执行工作项
GET    /v1/search                         — 搜索（LIKE）
GET    /v1/smart-search                   — 全文检索（FTS5 + BM25）
GET    /v1/events                         — 事件列表
POST   /v1/admin/sweep-leases             — 清理过期 lease
POST   /v1/admin/empty-trash              — 清空回收站
POST   /v1/admin/rebuild-index            — 重建全文检索索引
```
