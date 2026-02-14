import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/connections'
  },
  {
    path: '/connections',
    name: 'Connections',
    component: () => import('@/views/connections.vue'),
    meta: { title: '连接管理' }
  },
  {
    path: '/query/:id?',
    name: 'Query',
    component: () => import('@/views/query.vue'),
    meta: { title: 'SQL 查询' }
  },
  {
    path: '/tables/:id',
    name: 'Tables',
    component: () => import('@/views/tables.vue'),
    meta: { title: '数据浏览' }
  },
  {
    path: '/schema-editor',
    name: 'SchemaEditor',
    component: () => import('@/views/schema-editor.vue'),
    meta: { title: '表结构编辑器' }
  },
  {
    path: '/export/:id',
    name: 'Export',
    component: () => import('@/views/export.vue'),
    meta: { title: '数据导出' }
  },
  {
    path: '/monitor',
    name: 'Monitor',
    component: () => import('@/views/monitor.vue'),
    meta: { title: '监控' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'DBM'} - Database Manager`
  next()
})

export default router
