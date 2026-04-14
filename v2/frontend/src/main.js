import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import naive from 'naive-ui'
import App from './App.vue'

const TaskList = () => import('./pages/TaskList.vue')
const TaskDetail = () => import('./pages/TaskDetail.vue')
const WorkPage = () => import('./pages/WorkPage.vue')
const SearchPage = () => import('./pages/SearchPage.vue')
const TrashPage = () => import('./pages/TrashPage.vue')
const DocsPage = () => import('./pages/DocsPage.vue')

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: TaskList, meta: { key: 'overview' } },
    { path: '/projects/:projectId', component: TaskList, props: true, meta: { key: 'overview' } },
    { path: '/tasks/:id', component: TaskDetail, props: true },
    { path: '/work', component: WorkPage, meta: { key: 'work' } },
    { path: '/search', component: SearchPage, meta: { key: 'search' } },
    { path: '/trash', component: TrashPage, meta: { key: 'trash' } },
    { path: '/docs', component: DocsPage, meta: { key: 'docs' } },
  ],
})

const app = createApp(App)
app.use(router)
app.use(naive)
app.mount('#app')
