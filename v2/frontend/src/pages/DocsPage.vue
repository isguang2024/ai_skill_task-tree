<template>
  <n-scrollbar style="max-height:calc(100vh - 84px);">
    <div style="max-width:1120px;margin:0 auto;padding:4px 0 24px;">
      <n-card size="small" style="margin-bottom:12px;">
        <template #header>
          <n-space align="center" :size="8">
            <n-tag type="info" size="small">文档中心</n-tag>
            <n-text strong>Task Tree V2 使用说明</n-text>
          </n-space>
        </template>
        <div style="font-size:14px;line-height:1.8;white-space:pre-wrap;">
          这里用于快速理解 Task Tree V2 的用途、入口、能力边界和使用方法。
          如果你是第一次接触本项目，建议先看这页，再进入任务、工作池或具体节点页面。
        </div>
      </n-card>

      <n-grid :cols="24" :x-gap="12" :y-gap="12">
        <n-gi :span="12">
          <n-card size="small" title="项目用处">
            <n-list size="small">
              <n-list-item>管理跨会话、跨阶段的任务树。</n-list-item>
              <n-list-item>把任务拆成节点、阶段、Run 和 Memory，方便追踪执行过程。</n-list-item>
              <n-list-item>同时提供前端工作台、HTTP API 和 MCP 接口。</n-list-item>
              <n-list-item>适合需要持续推进、反复确认、可回溯的工作流。</n-list-item>
            </n-list>
          </n-card>
        </n-gi>
        <n-gi :span="12">
          <n-card size="small" title="访问入口">
            <n-list size="small">
              <n-list-item>工作台：<n-text code>http://127.0.0.1:8880</n-text></n-list-item>
              <n-list-item>HTTP API：<n-text code>http://127.0.0.1:8880/v1</n-text></n-list-item>
              <n-list-item>MCP：<n-text code>http://127.0.0.1:8880/mcp</n-text></n-list-item>
              <n-list-item>前端开发：<n-text code>http://127.0.0.1:5174</n-text></n-list-item>
            </n-list>
          </n-card>
        </n-gi>

        <n-gi :span="8">
          <n-card size="small" title="如何使用">
            <n-list size="small">
              <n-list-item>先从任务总览或工作池进入具体任务。</n-list-item>
              <n-list-item>任务详情页可查看节点树、Run、Memory、事件和产物。</n-list-item>
              <n-list-item>需要执行时，优先领取节点，再上报进度、开始 Run、结束 Run。</n-list-item>
              <n-list-item>保存前先看按钮悬浮提示，避免误操作。</n-list-item>
            </n-list>
          </n-card>
        </n-gi>
        <n-gi :span="8">
          <n-card size="small" title="能力分类">
            <n-list size="small">
              <n-list-item>任务：创建、列表、更新、流转、回收。</n-list-item>
              <n-list-item>节点：创建、领取、进度、完成、阻塞、重开。</n-list-item>
              <n-list-item>阶段：创建、激活、查看阶段 Memory。</n-list-item>
              <n-list-item>Run：开始、结束、日志、历史查看。</n-list-item>
            </n-list>
          </n-card>
        </n-gi>
        <n-gi :span="8">
          <n-card size="small" title="Memory 规则">
            <n-list size="small">
              <n-list-item>人工备注：`manual_note_text`。</n-list-item>
              <n-list-item>结构化摘要：`summary_text`、`conclusions`、`decisions`、`risks`、`blockers`、`next_actions`、`evidence`。</n-list-item>
              <n-list-item>完成节点或阶段推进后，优先刷新对应 Memory。</n-list-item>
            </n-list>
          </n-card>
        </n-gi>

        <n-gi :span="12">
          <n-card size="small" title="Run 是什么">
            <n-text depth="3" style="font-size:13px;line-height:1.8;white-space:pre-wrap;">
              Run 是节点的一次执行记录。它会保存输入摘要、输出摘要、执行者、开始结束时间和日志流。
              用 Run 的目的不是替代节点本身，而是给“这次执行到底做了什么”留一个可追踪的过程记录。
              常见场景是：开始执行前创建 Run，执行过程中追加日志，结束时写入结果。
            </n-text>
          </n-card>
        </n-gi>
        <n-gi :span="12">
          <n-card size="small" title="升级与排查">
            <n-list size="small">
              <n-list-item>V1 / Core：`8879` 和 `5173`。</n-list-item>
              <n-list-item>V2：`8880` 和 `5174`。</n-list-item>
              <n-list-item>技能与 DXT 都要指向 V2 的 `8880`。</n-list-item>
              <n-list-item>如果页面打不开，先检查后端是否启动和 `healthz` 是否可访问。</n-list-item>
            </n-list>
          </n-card>
        </n-gi>
      </n-grid>
    </div>
  </n-scrollbar>
</template>

<script setup>
</script>
