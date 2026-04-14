<template>
  <template v-if="selectedNode">
    <n-card size="small" style="margin-bottom:12px;">
      <template #header>
        <n-space align="center" :size="6">
          <n-tooltip>
            <template #trigger>
              <n-tag :type="statusType(selectedNode.status)" :bordered="false">{{ stateLabel(selectedNode.status, selectedNode.result) }}</n-tag>
            </template>
            <span>节点当前状态；决定是否可领取、完成或流转</span>
          </n-tooltip>
          <n-tag size="small">{{ selectedNode.kind === 'group' ? '分组' : '叶子' }}</n-tag>
          <n-tag type="info" size="small">{{ nodePct(selectedNode) }}%</n-tag>
          <n-tag v-if="isNodeClaimed" type="success" size="small">已领取 {{ claimedBy }}</n-tag>
        </n-space>
      </template>
      <template #header-extra>
        <n-space :size="4">
          <n-tooltip>
            <template #trigger>
              <n-button size="tiny" quaternary @click="toggleFav(selectedNode)">{{ isFav ? '★' : '☆' }}</n-button>
            </template>
            <span>收藏或取消收藏当前节点</span>
          </n-tooltip>
          <n-tooltip>
            <template #trigger>
              <n-button size="tiny" quaternary @click="copyId(selectedNode.id)">复制 ID</n-button>
            </template>
            <span>复制节点 ID 便于外部引用</span>
          </n-tooltip>
        </n-space>
      </template>

      <div style="font-size:16px;font-weight:600;margin-bottom:2px;">{{ selectedNode.title }}</div>
      <n-text depth="3" style="font-size:12px;">{{ selectedNode.path }}</n-text>
      <n-progress :percentage="nodePct(selectedNode)" :height="6" :border-radius="3" style="margin:10px 0;" />

      <n-space :size="6" style="margin-bottom:12px;">
        <n-tooltip v-if="canClaim"><template #trigger><n-button size="small" type="primary" @click="confirmClaimNode">领取</n-button></template><span>领取当前节点的执行权</span></n-tooltip>
        <n-tooltip v-if="canRelease"><template #trigger><n-button size="small" @click="confirmReleaseNode">释放</n-button></template><span>归还当前节点的执行权</span></n-tooltip>
        <n-tooltip v-if="canProgress"><template #trigger><n-button size="small" type="info" @click="confirmOpenProgressModal">上报进度</n-button></template><span>记录当前节点进展</span></n-tooltip>
        <n-tooltip v-if="canComplete"><template #trigger><n-button size="small" type="success" @click="confirmOpenCompleteModal">完成</n-button></template><span>将当前节点标记完成</span></n-tooltip>
        <n-tooltip v-if="canBlock"><template #trigger><n-button size="small" type="error" ghost @click="confirmBlockNode">阻塞</n-button></template><span>将节点标记为阻塞</span></n-tooltip>
        <n-tooltip v-if="canUnblock"><template #trigger><n-button size="small" @click="confirmNodeTransition('unblock')">解除阻塞</n-button></template><span>恢复节点为可执行状态</span></n-tooltip>
        <n-tooltip v-if="canPause"><template #trigger><n-button size="small" type="warning" ghost @click="confirmNodeTransition('pause')">暂停</n-button></template><span>暂停当前节点</span></n-tooltip>
        <n-tooltip v-if="canReopen"><template #trigger><n-button size="small" @click="confirmNodeTransition('reopen')">重开</n-button></template><span>把已关闭节点重新打开</span></n-tooltip>
        <n-tooltip v-if="canCancel"><template #trigger><n-button size="small" type="error" ghost @click="confirmNodeTransition('cancel')">取消</n-button></template><span>取消当前节点</span></n-tooltip>
        <n-tooltip v-if="canConvertToLeaf"><template #trigger><n-button size="small" @click="confirmRetypeNode">转回执行节点</n-button></template><span>把无子节点的分组改回执行节点</span></n-tooltip>
        <n-tooltip><template #trigger><n-button size="small" quaternary @click="confirmOpenNodeCreate(selectedNode.id)">添加子节点</n-button></template><span>在当前节点下新增子节点</span></n-tooltip>
        <n-tooltip v-if="selectedNode.parent_node_id"><template #trigger><n-button size="small" quaternary @click="confirmOpenNodeCreate(selectedNode.parent_node_id)">添加同级</n-button></template><span>在同级位置新增兄弟节点</span></n-tooltip>
        <n-tooltip v-if="!activeRun"><template #trigger><n-button size="small" type="info" ghost @click="confirmOpenRunStart">开始 Run</n-button></template><span>为该节点创建一次运行记录</span></n-tooltip>
        <n-tooltip v-if="activeRun"><template #trigger><n-button size="small" type="success" ghost @click="confirmOpenRunFinish">结束 Run</n-button></template><span>结束当前运行记录</span></n-tooltip>
        <n-tooltip v-if="activeRun"><template #trigger><n-button size="small" quaternary @click="confirmOpenRunLog">追加日志</n-button></template><span>向当前 Run 追加日志</span></n-tooltip>
      </n-space>

      <div v-if="selectedNode.instruction" style="margin-bottom:12px;">
        <n-text depth="3" style="font-size:12px;font-weight:600;">INSTRUCTION</n-text>
        <div style="font-size:13px;line-height:1.7;white-space:pre-wrap;margin-top:4px;padding:10px 12px;background:var(--n-color-modal);border-radius:4px;border:1px solid var(--n-border-color);">{{ selectedNode.instruction }}</div>
      </div>

      <n-collapse style="margin-bottom:0;">
        <n-collapse-item v-if="selectedNode.acceptance_criteria?.length" title="验收标准" name="acceptance">
          <n-list size="small" bordered>
            <n-list-item v-for="(c, i) in selectedNode.acceptance_criteria" :key="i">
              <n-text>{{ c }}</n-text>
            </n-list-item>
          </n-list>
        </n-collapse-item>
        <n-collapse-item v-if="parsedDependsOn.length" title="前置依赖" name="depends">
          <n-list size="small" bordered>
            <n-list-item v-for="depId in parsedDependsOn" :key="depId">
              <n-space align="center" :size="8">
                <n-tag :type="depNodeStatus(depId) === 'done' ? 'success' : depNodeStatus(depId) === 'canceled' ? 'default' : 'warning'" size="small">{{ depNodeStatus(depId) }}</n-tag>
                <n-text style="font-size:12px;cursor:pointer;" @click="selectNode(depId)">{{ depNodeTitle(depId) || depId }}</n-text>
              </n-space>
            </n-list-item>
          </n-list>
        </n-collapse-item>
        <n-collapse-item title="节点信息" name="info">
          <n-descriptions :column="2" label-placement="top" bordered size="small">
            <n-descriptions-item label="节点 ID">
              <n-text code style="font-size:11px;cursor:pointer;" @click="copyId(selectedNode.id)">{{ selectedNode.id }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="任务 ID">
              <n-text code style="font-size:11px;cursor:pointer;" @click="copyId(taskId)">{{ taskId }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="估时">{{ selectedNode.estimate || 0 }}h</n-descriptions-item>
            <n-descriptions-item label="类型">{{ selectedNode.kind === 'group' ? '分组节点' : '叶子节点' }}</n-descriptions-item>
            <n-descriptions-item label="路径">{{ selectedNode.path }}</n-descriptions-item>
            <n-descriptions-item v-if="selectedNode.lease_until" label="租约">{{ shortTime(selectedNode.lease_until) }}</n-descriptions-item>
          </n-descriptions>
        </n-collapse-item>
      </n-collapse>

      <n-grid :cols="2" :x-gap="8" style="margin-top:12px;">
        <n-gi>
          <n-card size="small" title="节点摘要">
            <div style="font-size:13px;line-height:1.7;white-space:pre-wrap;">{{ selectedMemoryText || '暂无节点摘要' }}</div>
            <n-text v-if="selectedNodeMemory && !selectedNodeMemory.execution_log" depth="3" style="font-size:11px;display:block;margin-top:6px;">完整执行日志请查看 Memory 标签页</n-text>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card size="small" title="节点运行历史">
            <n-empty v-if="nodeRuns.length === 0" description="暂无运行记录" size="small" />
            <n-list v-else size="small" hoverable clickable>
              <n-list-item
                v-for="run in nodeRuns"
                :key="run.id || run.run_id"
                @click="selectRun(run.id || run.run_id)"
                :style="selectedRunId === (run.id || run.run_id) ? 'background:var(--n-color-hover);border-radius:6px;' : ''"
              >
                <n-space vertical :size="4" style="width:100%;">
                  <n-space justify="space-between" style="width:100%;" align="center">
                    <n-space :size="4" align="center">
                      <n-tooltip>
                        <template #trigger>
                          <n-tag :type="run.status === 'running' ? 'info' : 'default'" size="small" :bordered="false">
                            {{ run.result || run.status || 'run' }}
                          </n-tag>
                        </template>
                        <span>Run 状态；running 表示执行中，其他值表示已结束</span>
                      </n-tooltip>
                      <n-text strong style="font-size:12px;">{{ run.trigger_kind || 'manual' }}</n-text>
                    </n-space>
                    <n-text depth="3" style="font-size:11px;">{{ shortTime(run.started_at || run.created_at) }}</n-text>
                  </n-space>
                  <n-space :size="8" wrap style="font-size:11px;">
                    <n-text depth="3">开始 {{ shortTime(run.started_at || run.created_at) }}</n-text>
                    <n-text depth="3">结束 {{ shortTime(run.finished_at || run.updated_at) || '进行中' }}</n-text>
                    <n-text depth="3">执行者 {{ formatRunActor(run) }}</n-text>
                  </n-space>
                </n-space>
              </n-list-item>
            </n-list>
          </n-card>
        </n-gi>
      </n-grid>
      <n-card v-if="selectedRun" size="small" title="Run 详情" style="margin-top:12px;">
        <template #header-extra>
          <n-space :size="4" align="center">
            <n-tooltip>
              <template #trigger>
                <n-tag size="small">{{ selectedRun.id || selectedRun.run_id }}</n-tag>
              </template>
              <span>Run 的唯一 ID</span>
            </n-tooltip>
            <n-tooltip>
              <template #trigger>
                <n-tag :type="selectedRun.status === 'running' ? 'info' : 'default'" size="small" :bordered="false">
                  {{ selectedRun.result || selectedRun.status || 'run' }}
                </n-tag>
              </template>
              <span>Run 的状态或结果</span>
            </n-tooltip>
          </n-space>
        </template>
        <n-spin :show="runDetailLoading">
          <n-grid :cols="2" :x-gap="8" style="margin-bottom:12px;">
            <n-gi>
              <n-card size="small" title="输入摘要">
                <div style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ selectedRun.input_summary || '无' }}</div>
              </n-card>
            </n-gi>
            <n-gi>
              <n-card size="small" title="输出摘要">
                <div style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ selectedRun.output_preview || selectedRun.error_text || '无' }}</div>
              </n-card>
            </n-gi>
          </n-grid>
          <n-space :size="8" wrap style="margin-bottom:12px;">
            <n-tooltip><template #trigger><n-tag size="small">触发 {{ selectedRun.trigger_kind || 'manual' }}</n-tag></template><span>Run 的触发来源</span></n-tooltip>
            <n-tooltip><template #trigger><n-tag size="small">开始 {{ shortTime(selectedRun.started_at || selectedRun.created_at) }}</n-tag></template><span>Run 开始执行的时间</span></n-tooltip>
            <n-tooltip><template #trigger><n-tag size="small">结束 {{ shortTime(selectedRun.finished_at || selectedRun.updated_at) || '进行中' }}</n-tag></template><span>Run 结束时间；未结束时显示进行中</span></n-tooltip>
            <n-tooltip><template #trigger><n-tag size="small">执行者 {{ formatRunActor(selectedRun) }}</n-tag></template><span>执行该 Run 的 actor 信息</span></n-tooltip>
          </n-space>
          <n-card size="small" title="日志流">
            <template #header-extra>
              <n-button size="tiny" quaternary @click="loadRunLogs()">
                {{ runLogsLoaded ? '刷新日志' : '加载日志' }}
              </n-button>
            </template>
            <n-empty v-if="!runLogsLoaded" description="按需加载日志" size="small" />
            <n-empty v-else-if="selectedRunLogs.length === 0" description="暂无日志" size="small" />
            <n-scrollbar v-else style="max-height:260px;">
              <n-list size="small">
                <n-list-item v-for="log in selectedRunLogs" :key="log.id || log.log_id">
                  <n-space vertical :size="4" style="width:100%;">
                    <n-space justify="space-between" style="width:100%;" align="center">
                      <n-space :size="4" align="center">
                        <n-tag size="small" type="info" :bordered="false">#{{ log.seq }}</n-tag>
                        <n-tag size="small">{{ log.kind || 'log' }}</n-tag>
                      </n-space>
                      <n-text depth="3" style="font-size:11px;">{{ shortTime(log.created_at) }}</n-text>
                    </n-space>
                    <div v-if="log.content" style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ log.content }}</div>
                    <div v-else-if="log.payload" style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ JSON.stringify(log.payload, null, 2) }}</div>
                  </n-space>
                </n-list-item>
              </n-list>
            </n-scrollbar>
          </n-card>
        </n-spin>
      </n-card>
    </n-card>

    <n-collapse style="margin-bottom:12px;">
      <n-collapse-item title="编辑节点" name="edit">
        <n-form :model="editForm" label-placement="top" size="small">
          <n-grid :cols="2" :x-gap="8">
            <n-gi>
              <n-form-item label="标题">
                <n-input v-model:value="editForm.title" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="估时(h)">
                <n-input-number v-model:value="editForm.estimate" :min="0" :step="0.5" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item label="Instruction">
            <n-input v-model:value="editForm.instruction" type="textarea" :rows="4" />
          </n-form-item>
          <n-form-item label="验收标准（一行一条）">
            <n-input v-model:value="editForm.acceptance" type="textarea" :rows="3" />
          </n-form-item>
          <n-form-item label="前置依赖（节点 ID，一行一个）">
            <n-input v-model:value="editForm.depends_on" type="textarea" :rows="2" placeholder="粘贴依赖节点的 ID，每行一个" />
          </n-form-item>
          <n-button type="primary" size="small" :loading="submitting" @click="saveNode">保存节点</n-button>
        </n-form>
      </n-collapse-item>
    </n-collapse>

    <n-card v-if="selectedChildren.length" size="small" title="子节点" style="margin-bottom:12px;">
      <template #header-extra>
        <n-space :size="4" align="center">
          <n-tag size="small">直接 {{ selectedChildren.length }}</n-tag>
          <n-tag v-if="selectedDescendantLeaves.length !== selectedChildren.length" size="small" type="info">叶子 {{ selectedDescendantLeaves.length }}</n-tag>
        </n-space>
      </template>
      <n-list size="small" hoverable clickable>
        <n-list-item v-for="child in selectedChildren" :key="child.id" @click="selectNode(child.id)">
          <template #prefix>
            <n-tooltip>
              <template #trigger>
                <n-tag :type="statusType(child.status)" size="small" :bordered="false">{{ stateLabel(child.status, child.result) }}</n-tag>
              </template>
              <span>子节点状态</span>
            </n-tooltip>
          </template>
          <div>
            <div style="font-size:13px;font-weight:500;">{{ child.title }}</div>
            <n-space :size="4" align="center" style="margin-top:2px;">
              <n-text depth="3" style="font-size:11px;">{{ child.path }}</n-text>
              <n-text code style="font-size:10px;cursor:pointer;color:var(--n-text-color-3);" @click.stop="copyId(child.id)">{{ child.id.substring(0, 15) }}…</n-text>
            </n-space>
          </div>
          <template #suffix>
            <n-space :size="4" align="center">
              <n-tag v-if="getChildCount(child.id) > 0" size="small" round>{{ getChildCount(child.id) }}</n-tag>
              <n-tag size="small" type="info">{{ nodePct(child) }}%</n-tag>
            </n-space>
          </template>
        </n-list-item>
      </n-list>
    </n-card>
  </template>
  <n-empty v-else description="选择左侧节点查看详情" />
</template>

<script setup>
defineProps({
  selectedNode: { type: Object, default: null },
  taskId: { type: String, required: true },
  statusType: { type: Function, required: true },
  stateLabel: { type: Function, required: true },
  nodePct: { type: Function, required: true },
  isNodeClaimed: { type: Boolean, default: false },
  claimedBy: { type: String, default: '' },
  isFav: { type: Boolean, default: false },
  toggleFav: { type: Function, required: true },
  copyId: { type: Function, required: true },
  canClaim: { type: Boolean, default: false },
  canRelease: { type: Boolean, default: false },
  canProgress: { type: Boolean, default: false },
  canComplete: { type: Boolean, default: false },
  canBlock: { type: Boolean, default: false },
  canUnblock: { type: Boolean, default: false },
  canPause: { type: Boolean, default: false },
  canReopen: { type: Boolean, default: false },
  canCancel: { type: Boolean, default: false },
  canConvertToLeaf: { type: Boolean, default: false },
  confirmClaimNode: { type: Function, required: true },
  confirmReleaseNode: { type: Function, required: true },
  confirmOpenProgressModal: { type: Function, required: true },
  confirmOpenCompleteModal: { type: Function, required: true },
  confirmBlockNode: { type: Function, required: true },
  confirmNodeTransition: { type: Function, required: true },
  confirmRetypeNode: { type: Function, required: true },
  confirmOpenNodeCreate: { type: Function, required: true },
  confirmOpenRunStart: { type: Function, required: true },
  confirmOpenRunFinish: { type: Function, required: true },
  confirmOpenRunLog: { type: Function, required: true },
  parsedDependsOn: { type: Array, default: () => [] },
  depNodeStatus: { type: Function, required: true },
  depNodeTitle: { type: Function, required: true },
  selectNode: { type: Function, required: true },
  shortTime: { type: Function, required: true },
  selectedMemoryText: { type: String, default: '' },
  selectedNodeMemory: { type: Object, default: null },
  nodeRuns: { type: Array, default: () => [] },
  selectedRunId: { type: String, default: '' },
  selectRun: { type: Function, required: true },
  formatRunActor: { type: Function, required: true },
  activeRun: { type: Object, default: null },
  selectedRun: { type: Object, default: null },
  runDetailLoading: { type: Boolean, default: false },
  runLogsLoaded: { type: Boolean, default: false },
  loadRunLogs: { type: Function, required: true },
  selectedRunLogs: { type: Array, default: () => [] },
  editForm: { type: Object, required: true },
  submitting: { type: Boolean, default: false },
  saveNode: { type: Function, required: true },
  selectedChildren: { type: Array, default: () => [] },
  selectedDescendantLeaves: { type: Array, default: () => [] },
  getChildCount: { type: Function, required: true },
})
</script>
