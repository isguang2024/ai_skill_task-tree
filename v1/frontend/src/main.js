import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import naive from 'naive-ui'
import App from './App.vue'
import TaskList from './pages/TaskList.vue'
import TaskDetail from './pages/TaskDetail.vue'
import WorkPage from './pages/WorkPage.vue'
import SearchPage from './pages/SearchPage.vue'
import TrashPage from './pages/TrashPage.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: TaskList, meta: { key: 'overview' } },
    { path: '/projects/:projectId', component: TaskList, props: true, meta: { key: 'overview' } },
    { path: '/tasks/:id', component: TaskDetail, props: true },
    { path: '/work', component: WorkPage, meta: { key: 'work' } },
    { path: '/search', component: SearchPage, meta: { key: 'search' } },
    { path: '/trash', component: TrashPage, meta: { key: 'trash' } },
  ],
})

const app = createApp(App)
app.use(router)
app.use(naive)
app.mount('#app')
