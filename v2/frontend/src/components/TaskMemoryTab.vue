<template>
  <n-collapse :default-expanded-names="['node']">
    <!-- ==================== 节点 Memory ==================== -->
    <n-collapse-item title="节点 Memory" name="node">
      <template v-if="selectedNode">
        <!-- 摘要 -->
        <div v-if="nodeMemorySummary" class="memory-summary">{{ nodeMemorySummary }}</div>

        <!-- 结构化字段（结论、决策、风险、证据等） -->
        <div v-if="nodeMemorySections.length" class="memory-sections">
          <div v-for="section in nodeMemorySections" :key="'node-' + section.key" class="memory-section">
            <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
            <n-list size="small" bordered class="memory-list">
              <n-list-item v-for="(item, index) in section.items" :key="'node-' + section.key + '-' + index">
                <div class="memory-item">{{ formatMemoryItem(item) }}</div>
              </n-list-item>
            </n-list>
          </div>
        </div>

        <!-- 分隔线 -->
        <n-divider v-if="nodeMemorySections.length || nodeMemorySummary" style="margin:8px 0;" />

        <!-- execution_log 按需加载区块 -->
        <div class="execution-log-section">
          <div class="execution-log-header" @click="toggleNodeExecLog">
            <n-space align="center" :size="8">
              <n-icon :component="nodeExecLogExpanded ? ChevronDown : ChevronRight" :size="14" />
              <n-text strong style="font-size:13px;">执行过程日志</n-text>
              <n-tag v-if="nodeExecLogContent" size="tiny" type="info" round>{{ lineCount(nodeExecLogContent) }} 行</n-tag>
              <n-tag v-else-if="nodeExecLogLoading" size="tiny" round>加载中...</n-tag>
              <n-tag v-else size="tiny" round>点击加载</n-tag>
            </n-space>
          </div>
          <div v-if="nodeExecLogExpanded" class="execution-log-body">
            <n-spin v-if="nodeExecLogLoading" size="small" style="display:block;margin:12px auto;" />
            <n-empty v-else-if="!nodeExecLogContent" description="暂无执行日志" size="small" style="margin:8px 0;" />
            <div v-else class="execution-log-content" :style="{ maxHeight: execLogMaxHeight + 'px' }">{{ nodeExecLogContent }}</div>
            <div v-if="nodeExecLogContent" class="execution-log-controls">
              <n-text depth="3" style="font-size:11px;">高度</n-text>
              <n-slider v-model:value="execLogMaxHeight" :min="200" :max="1200" :step="100" style="width:120px;" />
              <n-text depth="3" style="font-size:11px;">{{ execLogMaxHeight }}px</n-text>
            </div>
          </div>
        </div>

        <n-divider style="margin:8px 0;" />
        <n-input v-model:value="memoryForm.node_note" type="textarea" :rows="2" placeholder="当前节点备注" />
        <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="emit('save-note', 'node')">保存节点备注</n-button>
      </template>
      <n-empty v-else description="请先选择节点" size="small" />
    </n-collapse-item>

    <!-- ==================== 阶段 Memory ==================== -->
    <n-collapse-item title="阶段 Memory" name="stage">
      <template v-if="currentStage">
        <div v-if="stageMemorySummary" class="memory-summary">{{ stageMemorySummary }}</div>
        <div v-if="stageMemorySections.length" class="memory-sections">
          <div v-for="section in stageMemorySections" :key="'stage-' + section.key" class="memory-section">
            <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
            <n-list size="small" bordered class="memory-list">
              <n-list-item v-for="(item, index) in section.items" :key="'stage-' + section.key + '-' + index">
                <div class="memory-item">{{ formatMemoryItem(item) }}</div>
              </n-list-item>
            </n-list>
          </div>
        </div>
        <div v-if="currentStageMemory?.execution_log" class="execution-log-section">
          <n-collapse>
            <n-collapse-item name="stage-exec-log">
              <template #header>
                <n-space align="center" :size="8">
                  <n-text strong style="font-size:13px;">执行过程日志</n-text>
                  <n-tag size="tiny" type="info" round>{{ lineCount(currentStageMemory.execution_log) }} 行</n-tag>
                </n-space>
              </template>
              <div class="execution-log-content" style="max-height:500px;">{{ currentStageMemory.execution_log }}</div>
            </n-collapse-item>
          </n-collapse>
        </div>
        <n-divider style="margin:8px 0;" />
        <n-input v-model:value="memoryForm.stage_note" type="textarea" :rows="2" placeholder="当前阶段备注" />
        <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="emit('save-note', 'stage')">保存阶段备注</n-button>
      </template>
      <n-empty v-else description="尚未进入阶段模式" size="small" />
    </n-collapse-item>

    <!-- ==================== 任务 Memory ==================== -->
    <n-collapse-item title="任务 Memory" name="task">
      <div v-if="taskMemorySummary" class="memory-summary">{{ taskMemorySummary }}</div>
      <div v-if="taskMemorySections.length" class="memory-sections">
        <div v-for="section in taskMemorySections" :key="'task-' + section.key" class="memory-section">
          <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
          <n-list size="small" bordered class="memory-list">
            <n-list-item v-for="(item, index) in section.items" :key="'task-' + section.key + '-' + index">
              <div class="memory-item">{{ formatMemoryItem(item) }}</div>
            </n-list-item>
          </n-list>
        </div>
      </div>
      <div v-if="taskMemory?.execution_log" class="execution-log-section">
        <n-collapse>
          <n-collapse-item name="task-exec-log">
            <template #header>
              <n-space align="center" :size="8">
                <n-text strong style="font-size:13px;">执行过程日志</n-text>
                <n-tag size="tiny" type="info" round>{{ lineCount(taskMemory.execution_log) }} 行</n-tag>
              </n-space>
            </template>
            <div class="execution-log-content" style="max-height:500px;">{{ taskMemory.execution_log }}</div>
          </n-collapse-item>
        </n-collapse>
      </div>
      <n-divider style="margin:8px 0;" />
      <n-input v-model:value="memoryForm.task_note" type="textarea" :rows="2" placeholder="人工备注写入 task memory.manual_note_text" />
      <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="emit('save-note', 'task')">保存任务备注</n-button>
    </n-collapse-item>
  </n-collapse>
</template>

<script setup>
import { ref, watch, h } from 'vue'
import { fetchNodeExecutionLog } from '../api'

const ChevronRight = { render: () => h('span', { style: 'font-size:14px;' }, '\u25B6') }
const ChevronDown = { render: () => h('span', { style: 'font-size:14px;' }, '\u25BC') }

const props = defineProps({
  selectedNode: { type: Object, default: null },
  selectedNodeMemory: { type: Object, default: null },
  nodeMemorySummary: { type: String, default: '' },
  nodeMemorySections: { type: Array, default: () => [] },
  currentStage: { type: Object, default: null },
  currentStageMemory: { type: Object, default: null },
  stageMemorySummary: { type: String, default: '' },
  stageMemorySections: { type: Array, default: () => [] },
  taskMemory: { type: Object, default: null },
  taskMemorySummary: { type: String, default: '' },
  taskMemorySections: { type: Array, default: () => [] },
  memoryForm: { type: Object, required: true },
  submitting: { type: Boolean, default: false },
  formatMemoryItem: { type: Function, required: true },
})

const emit = defineEmits(['save-note'])

// execution_log lazy loading state
const nodeExecLogExpanded = ref(false)
const nodeExecLogLoading = ref(false)
const nodeExecLogContent = ref('')
const execLogMaxHeight = ref(400)

// Reset when node changes
watch(() => props.selectedNode?.id, () => {
  nodeExecLogExpanded.value = false
  nodeExecLogLoading.value = false
  nodeExecLogContent.value = ''
})

// If memory already has execution_log (from full preset), use it directly
watch(() => props.selectedNodeMemory?.execution_log, (val) => {
  if (val) nodeExecLogContent.value = val
})

async function toggleNodeExecLog() {
  nodeExecLogExpanded.value = !nodeExecLogExpanded.value
  if (nodeExecLogExpanded.value && !nodeExecLogContent.value && props.selectedNode?.id) {
    nodeExecLogLoading.value = true
    try {
      nodeExecLogContent.value = await fetchNodeExecutionLog(props.selectedNode.id)
    } catch { /* ignore */ }
    nodeExecLogLoading.value = false
  }
}

function lineCount(text) {
  if (!text) return 0
  return text.split('\n').filter(l => l.trim()).length
}
</script>
