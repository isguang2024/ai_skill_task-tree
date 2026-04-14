<template>
  <n-card size="small">
    <n-space align="center" justify="space-between" style="margin-bottom:10px;">
      <n-radio-group :value="eventScope" size="small" @update:value="value => emit('update:eventScope', value)">
        <n-radio-button value="all">全部事件 ({{ events.length }})</n-radio-button>
        <n-radio-button value="node" :disabled="!selectedNodeId">
          {{ selectedNodeId ? selectedNodeLabel + (nodeEvents.length ? ' (' + nodeEvents.length + ')' : '') : '未选节点' }}
        </n-radio-button>
      </n-radio-group>
      <n-checkbox
        v-if="eventScope === 'node'"
        :checked="eventsWarnOnly"
        size="small"
        @update:checked="value => emit('update:eventsWarnOnly', value)"
      >
        仅异常
      </n-checkbox>
    </n-space>

    <template v-if="eventScope === 'all'">
      <n-empty v-if="events.length === 0" description="暂无事件" />
      <n-scrollbar v-else style="max-height:calc(100vh - 360px);">
        <n-timeline>
          <n-timeline-item
            v-for="ev in events"
            :key="ev.id"
            :type="timelineType(ev.type)"
            :title="eventTypeLabel(ev.type)"
            :time="shortTime(ev.created_at)"
          >
            <n-space :size="4" align="center" style="margin-bottom:2px;">
              <n-text depth="3" style="font-size:11px;">{{ nodePath(ev) }}</n-text>
            </n-space>
            <div v-if="ev.message" style="font-size:12px;white-space:pre-wrap;color:var(--n-text-color-2);">{{ ev.message }}</div>
            <div v-if="ev.actor_type || ev.actor_id" style="font-size:11px;color:var(--n-text-color-3);">{{ (ev.actor_type || '') + ' ' + (ev.actor_id || '') }}</div>
          </n-timeline-item>
        </n-timeline>
      </n-scrollbar>
    </template>

    <template v-else>
      <n-empty v-if="!selectedNodeId" description="请先在左侧选择一个节点" />
      <n-empty v-else-if="filteredNodeEvents.length === 0" description="暂无事件" />
      <n-scrollbar v-else style="max-height:calc(100vh - 360px);">
        <n-timeline>
          <n-timeline-item
            v-for="ev in filteredNodeEvents"
            :key="ev.id"
            :type="timelineType(ev.type)"
            :title="eventTypeLabel(ev.type)"
            :time="shortTime(ev.created_at)"
          >
            <n-space :size="4" align="center" style="margin-bottom:2px;">
              <n-text depth="3" style="font-size:11px;">{{ nodePath(ev) }}</n-text>
            </n-space>
            <div v-if="ev.message" style="font-size:12px;white-space:pre-wrap;color:var(--n-text-color-2);">{{ ev.message }}</div>
            <div v-if="ev.actor_type || ev.actor_id" style="font-size:11px;color:var(--n-text-color-3);">{{ (ev.actor_type || '') + ' ' + (ev.actor_id || '') }}</div>
          </n-timeline-item>
        </n-timeline>
      </n-scrollbar>
    </template>
  </n-card>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  events: { type: Array, default: () => [] },
  eventScope: { type: String, default: 'all' },
  eventsWarnOnly: { type: Boolean, default: false },
  selectedNodeId: { type: String, default: '' },
  selectedNode: { type: Object, default: null },
  nodeEvents: { type: Array, default: () => [] },
  nodes: { type: Array, default: () => [] },
  eventTypeLabel: { type: Function, required: true },
  shortTime: { type: Function, required: true },
})

const emit = defineEmits(['update:eventScope', 'update:eventsWarnOnly'])

const selectedNodeLabel = computed(() => {
  const title = props.selectedNode?.title || '当前节点'
  return title.substring(0, 12)
})

const filteredNodeEvents = computed(() => {
  if (!props.eventsWarnOnly) return props.nodeEvents
  return props.nodeEvents.filter(ev => {
    const t = (ev.type || '').toLowerCase()
    return t === 'blocked' || t === 'reopened' || t === 'error' || ((ev.payload?.warnings || []).length > 0)
  })
})

function timelineType(type) {
  if (type === 'complete') return 'success'
  if (type === 'blocked') return 'error'
  return 'default'
}

function nodePath(ev) {
  if (!ev?.node_id) return ''
  return props.nodes.find(n => n.id === ev.node_id)?.path || ev.node_id.substring(0, 12)
}
</script>
