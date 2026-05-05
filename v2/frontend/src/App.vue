<template>
  <n-config-provider :theme="isDark ? darkTheme : null" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <GlobalApiProvider />

          <n-layout has-sider position="absolute" class="app-shell">
            <n-layout-sider v-if="showMixedPrimarySider" bordered :width="72" class="primary-sider">
              <div class="sider-top-spacer" />
              <n-menu
                :options="primarySiderOptions"
                :value="activeTopKey"
                :collapsed="true"
                :collapsed-width="72"
                :collapsed-icon-size="18"
                @update:value="onPrimarySiderSelect"
              />
            </n-layout-sider>

            <n-layout-sider
              v-if="showSecondarySider"
              bordered
              :width="secondarySiderWidth"
              collapse-mode="width"
              :collapsed-width="64"
              :collapsed="secondarySiderCollapsed"
              :show-trigger="secondarySiderCollapsible"
              @collapse="secondarySiderCollapsed = true"
              @expand="secondarySiderCollapsed = false"
            >
              <div class="sider-top-spacer" />
              <n-menu
                :options="secondarySiderOptions"
                :value="activeMenuPath"
                :collapsed="secondarySiderCollapsed"
                :collapsed-width="64"
                :expanded-keys="secondaryExpandedKeys"
                @update:value="onSecondaryMenuSelect"
              />
            </n-layout-sider>

            <n-layout class="app-main">
              <n-layout-header bordered class="app-header">
                <button class="brand" type="button" @click="router.push('/')">
                  <svg class="brand-logo" viewBox="0 0 32 32" aria-hidden="true">
                    <defs>
                      <linearGradient id="task-tree-brand" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" stop-color="#2080f0" />
                        <stop offset="100%" stop-color="#7c9cff" />
                      </linearGradient>
                    </defs>
                    <path d="M7 19.5 16 4l9 15.5H7Z" fill="url(#task-tree-brand)" opacity="0.95" />
                    <circle cx="10" cy="22.5" r="3" fill="#2080f0" opacity="0.9" />
                    <circle cx="22" cy="22.5" r="3" fill="#18a058" opacity="0.9" />
                  </svg>
                  <span class="brand-name">Task Tree</span>
                </button>

                <n-menu
                  v-if="showHeaderMenu"
                  class="header-menu"
                  mode="horizontal"
                  :options="headerMenuOptions"
                  :value="activeTopKey"
                  @update:value="onHeaderMenuSelect"
                />

                <div class="header-actions">
                  <n-select
                    v-model:value="layoutMode"
                    class="layout-mode-select"
                    :options="layoutModeOptions"
                    size="small"
                    placeholder="菜单模式"
                  />
                  <n-button size="small" type="primary" @click="showCreateTask = true">新建任务</n-button>
                  <n-button size="small" quaternary @click="isDark = !isDark">{{ isDark ? '☀️' : '🌙' }}</n-button>
                  <n-button v-if="aiEnabled" size="small" quaternary @click="showAI = !showAI">AI 助手</n-button>
                </div>
              </n-layout-header>

              <n-layout-content content-style="padding:16px;background:var(--n-color);" :native-scrollbar="false">
                <div class="content-breadcrumb">
                  <n-breadcrumb>
                    <n-breadcrumb-item
                      v-for="(item, i) in breadcrumbItems"
                      :key="i"
                      :style="item.path && i < breadcrumbItems.length - 1 ? 'cursor:pointer;' : ''"
                      @click="item.path ? router.push(item.path) : undefined"
                    >
                      {{ item.label }}
                    </n-breadcrumb-item>
                  </n-breadcrumb>
                </div>
                <router-view />
              </n-layout-content>
            </n-layout>
          </n-layout>

          <n-modal v-model:show="showCreateTask" preset="card" title="新建任务" style="max-width:560px;">
            <n-form ref="createFormRef" :model="createForm" label-placement="top">
              <n-form-item label="任务标题" path="title" :rule="{ required: true, message: '请输入标题' }">
                <n-input v-model:value="createForm.title" placeholder="例如：重构订单同步链路" />
              </n-form-item>
              <n-form-item label="任务 Key（可选）" path="task_key">
                <n-input v-model:value="createForm.task_key" placeholder="例如：SYNC" />
              </n-form-item>
              <n-form-item label="任务目标 (Goal)" path="goal">
                <n-input
                  v-model:value="createForm.goal"
                  type="textarea"
                  :rows="4"
                  placeholder="2-4句，说清交付标准、约束和范围外项"
                />
              </n-form-item>
            </n-form>
            <template #action>
              <n-button @click="showCreateTask = false">取消</n-button>
              <n-button type="primary" :loading="creating" @click="doCreateTask">创建任务</n-button>
            </template>
          </n-modal>

          <n-drawer v-model:show="showAI" :width="420" placement="right">
            <n-drawer-content title="AI 工作助手" :native-scrollbar="false">
              <template #header-extra>
                <n-space>
                  <n-select
                    v-model:value="aiModel"
                    :options="aiModels"
                    size="small"
                    style="width:160px;"
                    placeholder="默认模型"
                    clearable
                  />
                  <n-button size="small" quaternary @click="clearAI">清空</n-button>
                </n-space>
              </template>

              <div ref="aiMessagesRef" style="min-height:200px;">
                <div v-if="aiMessages.length === 0" style="color:var(--n-text-color-3);padding:20px;">
                  可以直接让我分析当前任务、补节点结构，或者推进节点操作。
                </div>
                <div v-for="(msg, i) in aiMessages" :key="i" style="margin-bottom:12px;">
                  <n-tag :type="msg.role === 'user' ? 'info' : 'success'" size="small">
                    {{ msg.role === 'user' ? '你' : 'AI' }}
                  </n-tag>
                  <div style="white-space:pre-wrap;margin-top:4px;font-size:13px;">{{ msg.content }}</div>
                </div>
                <div v-if="aiLoading" style="padding:8px 0;">
                  <n-spin size="small" /> 思考中...
                </div>
              </div>

              <template #footer>
                <n-input-group>
                  <n-input
                    v-model:value="aiInput"
                    type="textarea"
                    :autosize="{ minRows: 1, maxRows: 4 }"
                    placeholder="向 AI 提问或下达操作指令…"
                    @keydown.enter.ctrl="sendAI"
                  />
                  <n-button type="primary" :loading="aiLoading" style="align-self:flex-end;" @click="sendAI">发送</n-button>
                </n-input-group>
              </template>
            </n-drawer-content>
          </n-drawer>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { ref, computed, watch, onMounted, h, nextTick, defineComponent, provide } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { darkTheme, useMessage, useDialog, useNotification } from 'naive-ui'
import { api } from './api.js'

const GlobalApiProvider = defineComponent({
  setup() {
    window.$message = useMessage()
    window.$dialog = useDialog()
    window.$notification = useNotification()
    return () => null
  },
})

const router = useRouter()
const route = useRoute()

const breadcrumbItems = ref([{ label: '任务总览', path: '/' }])
provide('breadcrumb', breadcrumbItems)

const themeOverrides = {
  common: {
    primaryColor: '#2080f0',
    primaryColorHover: '#4098fc',
    primaryColorPressed: '#1060c9',
    primaryColorSuppl: '#4098fc',
  },
}

const LAYOUT_STORAGE_KEY = 'task-tree.layout.mode'
const VALID_LAYOUT_MODES = [
  'sidebar-nav',
  'sidebar-mixed-nav',
  'header-nav',
  'header-sidebar-nav',
  'mixed-nav',
  'header-mixed-nav',
]

const layoutModeOptions = [
  { label: '侧边菜单 sidebar-nav', value: 'sidebar-nav' },
  { label: '双列侧边 sidebar-mixed-nav', value: 'sidebar-mixed-nav' },
  { label: '顶部菜单 header-nav', value: 'header-nav' },
  { label: '顶部+侧边 header-sidebar-nav', value: 'header-sidebar-nav' },
  { label: '混合菜单 mixed-nav', value: 'mixed-nav' },
  { label: '头部双列 header-mixed-nav', value: 'header-mixed-nav' },
]

const menuTree = [
  {
    key: 'overview',
    label: '概览',
    icon: 'overview',
    path: '/',
    children: [
      { key: 'overview-home', label: '任务总览', icon: 'overview', path: '/' },
      { key: 'overview-docs', label: '文档', icon: 'docs', path: '/docs' },
    ],
  },
  {
    key: 'work',
    label: '工作台',
    icon: 'work',
    path: '/work',
    children: [
      { key: 'work-pool', label: '工作池', icon: 'work', path: '/work' },
      { key: 'work-search', label: '搜索', icon: 'search', path: '/search' },
    ],
  },
  {
    key: 'manage',
    label: '管理',
    icon: 'manage',
    path: '/trash',
    children: [{ key: 'manage-trash', label: '回收站', icon: 'trash', path: '/trash' }],
  },
]

const layoutMode = ref('sidebar-nav')
const secondarySiderCollapsed = ref(false)
const isDark = ref(false)
const showCreateTask = ref(false)
const creating = ref(false)
const createForm = ref({ title: '', task_key: '', goal: '' })
const createFormRef = ref(null)

const aiEnabled = ref(false)
const showAI = ref(false)
const aiInput = ref('')
const aiMessages = ref([])
const aiLoading = ref(false)
const aiModel = ref(null)
const aiMessagesRef = ref(null)

const aiModels = [
  { label: 'gpt-5.4', value: 'gpt-5.4' },
  { label: 'gpt-5.3-codex', value: 'gpt-5.3-codex' },
  { label: 'gpt-4o', value: 'gpt-4o' },
  { label: 'claude-opus-4-5', value: 'claude-opus-4-5' },
  { label: 'claude-sonnet-4-5', value: 'claude-sonnet-4-5' },
]

const showHeaderMenu = computed(() =>
  ['header-nav', 'header-sidebar-nav', 'mixed-nav', 'header-mixed-nav'].includes(layoutMode.value),
)

const showMixedPrimarySider = computed(() =>
  ['sidebar-mixed-nav', 'header-mixed-nav'].includes(layoutMode.value),
)

const showSecondarySider = computed(() =>
  ['sidebar-nav', 'header-sidebar-nav', 'mixed-nav', 'sidebar-mixed-nav', 'header-mixed-nav'].includes(
    layoutMode.value,
  ),
)

const secondarySiderCollapsible = computed(() =>
  ['sidebar-nav', 'header-sidebar-nav', 'mixed-nav'].includes(layoutMode.value),
)

const secondarySiderWidth = computed(() => (showMixedPrimarySider.value ? 208 : 224))

const isFullTreeSideMode = computed(() => ['sidebar-nav', 'header-sidebar-nav'].includes(layoutMode.value))

const activeTopKey = computed(() => resolveTopKeyByPath(route.path))

const activeTopMenu = computed(() => {
  return menuTree.find((item) => item.key === activeTopKey.value) || menuTree[0]
})

const headerMenuOptions = computed(() =>
  menuTree.map((item) => ({
    key: item.key,
    label: item.label,
    icon: renderMenuIcon(item.icon),
  })),
)

const primarySiderOptions = computed(() =>
  menuTree.map((item) => ({
    key: item.key,
    label: item.label,
    icon: renderMenuIcon(item.icon),
  })),
)

const mixedChildrenOptions = computed(() =>
  (activeTopMenu.value?.children || []).map((item) => ({
    key: item.path,
    label: item.label,
    icon: renderMenuIcon(item.icon),
  })),
)

const fullTreeOptions = computed(() =>
  menuTree.map((group) => ({
    key: group.path,
    label: group.label,
    icon: renderMenuIcon(group.icon),
    children: (group.children || []).map((child) => ({
      key: child.path,
      label: child.label,
      icon: renderMenuIcon(child.icon),
    })),
  })),
)

const secondarySiderOptions = computed(() => {
  if (isFullTreeSideMode.value) return fullTreeOptions.value
  return mixedChildrenOptions.value
})

const secondaryExpandedKeys = computed(() => {
  if (!isFullTreeSideMode.value) return []
  return [activeTopMenu.value.path]
})

const activeMenuPath = computed(() => {
  const normalized = normalizeMenuPath(route.path)
  if (isPathInMenu(normalized)) return normalized
  const first = activeTopMenu.value?.children?.[0]?.path
  return first || activeTopMenu.value?.path || '/'
})

watch(layoutMode, (mode) => {
  localStorage.setItem(LAYOUT_STORAGE_KEY, mode)
  if (!secondarySiderCollapsible.value) secondarySiderCollapsed.value = false
})

watch(secondarySiderCollapsible, (enabled) => {
  if (!enabled) secondarySiderCollapsed.value = false
})

function resolveTopKeyByPath(path) {
  if (path === '/work' || path.startsWith('/search')) return 'work'
  if (path.startsWith('/trash')) return 'manage'
  return 'overview'
}

function normalizeMenuPath(path) {
  if (path.startsWith('/projects/')) return '/'
  if (path.startsWith('/tasks/')) return '/'
  return path
}

function isPathInMenu(path) {
  return menuTree.some((group) => {
    if (group.path === path) return true
    return (group.children || []).some((child) => child.path === path)
  })
}

function onHeaderMenuSelect(key) {
  const item = menuTree.find((menu) => menu.key === key)
  if (item?.path) router.push(item.path)
}

function onPrimarySiderSelect(key) {
  const item = menuTree.find((menu) => menu.key === key)
  if (item?.path) router.push(item.path)
}

function onSecondaryMenuSelect(key) {
  if (typeof key === 'string' && key.startsWith('/')) {
    router.push(key)
  }
}

function renderMenuIcon(name) {
  const paths = {
    overview: 'M4 6h16M4 12h16M4 18h16',
    work: 'M4 7h6l2 2h8v10H4z',
    search: 'M11 4a7 7 0 1 0 4.24 12.6l4.08 4.09 1.42-1.42-4.09-4.08A7 7 0 0 0 11 4z',
    docs: 'M6 4h9l3 3v13H6zM9 9h6M9 13h6M9 17h4',
    manage: 'M4 6h16v12H4zM8 10h8M8 14h5',
    trash: 'M6 7h12M9 7V5h6v2m-7 0 .7 11h6.6L16 7',
  }
  return () =>
    h(
      'svg',
      {
        viewBox: '0 0 24 24',
        width: '18',
        height: '18',
        fill: 'none',
        stroke: 'currentColor',
        'stroke-width': '1.8',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        style: 'display:block;',
      },
      [h('path', { d: paths[name] || paths.overview })],
    )
}

async function doCreateTask() {
  if (!createForm.value.title.trim()) return
  creating.value = true
  try {
    const body = { title: createForm.value.title }
    if (createForm.value.task_key) body.task_key = createForm.value.task_key
    if (createForm.value.goal) body.goal = createForm.value.goal
    const result = await api('/tasks', { method: 'POST', body: JSON.stringify(body) })
    showCreateTask.value = false
    createForm.value = { title: '', task_key: '', goal: '' }
    router.push('/tasks/' + result.id)
  } catch (e) {
    window.$message?.error('创建失败: ' + e.message)
  } finally {
    creating.value = false
  }
}

async function checkAI() {
  try {
    const res = await fetch('/ai/status')
    const data = await res.json()
    aiEnabled.value = data.enabled === true
  } catch {
    aiEnabled.value = false
  }
}

async function sendAI() {
  const text = aiInput.value.trim()
  if (!text || aiLoading.value) return
  aiMessages.value.push({ role: 'user', content: text })
  aiInput.value = ''
  aiLoading.value = true
  try {
    const body = { message: text }
    if (aiModel.value) body.model = aiModel.value
    if (route.params.id) body.task_id = route.params.id
    const res = await fetch('/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({ detail: res.statusText }))
      throw new Error(err.detail || err.error || res.statusText)
    }
    const data = await res.json()
    aiMessages.value.push({ role: 'assistant', content: data.reply || data.message || JSON.stringify(data) })
  } catch (e) {
    aiMessages.value.push({ role: 'assistant', content: '请求失败: ' + e.message })
  } finally {
    aiLoading.value = false
    await nextTick()
    if (aiMessagesRef.value) aiMessagesRef.value.scrollTop = aiMessagesRef.value.scrollHeight
  }
}

async function clearAI() {
  try {
    await fetch('/ai/clear', { method: 'POST' })
  } catch {}
  aiMessages.value = []
}

onMounted(() => {
  const saved = localStorage.getItem(LAYOUT_STORAGE_KEY)
  if (saved && VALID_LAYOUT_MODES.includes(saved)) layoutMode.value = saved
  checkAI()
})
</script>

<style scoped>
.app-shell {
  height: 100vh;
}

.primary-sider {
  border-right: 1px solid var(--n-border-color);
}

.app-main {
  min-width: 0;
}

.app-header {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 56px;
  padding: 0 12px;
  flex-wrap: nowrap;
  min-width: 0;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  border: 0;
  background: transparent;
  color: inherit;
  padding: 0;
  cursor: pointer;
  white-space: nowrap;
  flex: 0 0 auto;
}

.brand-logo {
  width: 28px;
  height: 28px;
  flex: none;
  display: block;
}

.brand-name {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.2px;
}

.header-menu {
  flex: 1 1 auto;
  min-width: 280px;
  overflow: hidden;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  white-space: nowrap;
}

.layout-mode-select {
  width: 224px;
}

.content-breadcrumb {
  margin-bottom: 12px;
}

.sider-top-spacer {
  height: 8px;
}

@media (max-width: 1280px) {
  .layout-mode-select {
    width: 176px;
  }
}
</style>
