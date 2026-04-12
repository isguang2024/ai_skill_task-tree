<template>
  <n-spin :show="loading">
    <n-card size="small" style="margin-bottom:12px;">
      <n-space align="center">
        <n-select v-model:value="statusFilter" :options="statusOpts" style="width:160px;"
          @update:value="loadWork" placeholder="状态筛选" />
        <n-switch v-model:value="includeClaimed" @update:value="loadWork">
          <template #checked>含已领取</template>
          <template #unchecked>仅未领取</template>
        </n-switch>
        <n-tag type="info" size="small">共 {{ items.length }} 项</n-tag>
      </n-space>
    </n-card>

    <n-empty v-if="!loading && items.length===0" description="暂无可领取工作项" style="padding:60px 0;" />

    <div v-for="item in items" :key="item.id" style="margin-bottom:8px;">
      <n-card hoverable size="small" style="cursor:pointer;" @click="goNode(item)">
        <template #header>
          <n-space align="center" :size="6">
            <n-tag :type="statusType(item.status)" size="small" :bordered="false">{{ statusLabel(item.status) }}</n-tag>
            <n-text depth="3" style="font-size:12px;">{{ item.path }}</n-text>
          </n-space>
        </template>
        <template #header-extra>
          <n-text depth="3" style="font-size:12px;">{{ shortTime(item.updated_at) }}</n-text>
        </template>
        <div style="font-size:15px;font-weight:600;margin-bottom:4px;">{{ item.title }}</div>
        <n-space :size="4" align="center">
          <n-tag size="small" round type="info">{{ item.task_title || '未知任务' }}</n-tag>
          <n-tag v-if="item.kind" size="small" round>{{ item.kind }}</n-tag>
          <n-tag v-if="leaseActive(item)" size="small" round type="warning">已领取</n-tag>
        </n-space>
      </n-card>
    </div>
  </n-spin>
</template>

<script setup>
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { api, statusType, statusLabel, shortTime, leaseActive } from '../api.js'

const router = useRouter()
const items = ref([])
const loading = ref(true)
const statusFilter = ref('ready')
const includeClaimed = ref(true)

const statusOpts = [
  { label: '就绪', value: 'ready' },
  { label: '进行中', value: 'running' },
  { label: '就绪+进行中', value: 'ready,running' },
]

async function loadWork() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (statusFilter.value) params.set('status', statusFilter.value)
    params.set('include_claimed', includeClaimed.value ? 'true' : 'false')
    params.set('limit', '60')
    items.value = await api('/work-items?' + params)
  } catch (e) {
    window.$message?.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

function goNode(item) {
  router.push('/tasks/' + item.task_id + '?node=' + item.id)
}

onMounted(loadWork)
onActivated(loadWork)
</script>
