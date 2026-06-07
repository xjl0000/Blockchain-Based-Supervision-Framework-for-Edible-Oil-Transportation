import Vue from 'vue'
import Router from 'vue-router'
import Layout from '@/layout'

Vue.use(Router)
const all = ['原料供应商', '榨油厂', '运输人员', '零售商', '监管机构', '系统管理员']
const businessRoles = ['原料供应商', '榨油厂', '运输人员', '零售商', '监管机构']

export const constantRoutes = [
  { path: '/login', component: () => import('@/views/login/index'), hidden: true },
  { path: '/404', component: () => import('@/views/404'), hidden: true },
  { path: '/forbidden', component: Layout, hidden: true, children: [
    { path: '', name: 'Forbidden', component: () => import('@/views/forbidden/index'), meta: { title: '权限提示' }}
  ] },
  { path: '/', component: Layout, redirect: '/dashboard', children: [
    { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/dashboard/index'), meta: { title: '主界面', icon: 'dashboard', roles: all }}
  ] },
  { path: '/raw-material', component: Layout, children: [
    { path: 'index', name: 'RawMaterial', component: () => import('@/views/batches/index'), meta: { title: '原材料管理', icon: 'form', roles: ['原料供应商'] }}
  ] },
  { path: '/pressing', component: Layout, children: [
    { path: 'index', name: 'Pressing', component: () => import('@/views/pressing/index'), meta: { title: '食用油压榨管理', icon: 'el-icon-office-building', roles: ['榨油厂'] }}
  ] },
  { path: '/transport', component: Layout, children: [
    { path: 'index', name: 'Transport', component: () => import('@/views/transport/index'), meta: { title: '运输管理', icon: 'el-icon-truck', roles: ['运输人员'] }}
  ] },
  { path: '/retail', component: Layout, children: [
    { path: 'index', name: 'Retail', component: () => import('@/views/retail/index'), meta: { title: '零售管理', icon: 'el-icon-shopping-bag-2', roles: ['零售商'] }}
  ] },
  { path: '/trace', component: Layout, children: [
    { path: 'index', name: 'Trace', component: () => import('@/views/trace/index'), meta: { title: '产品全流程追溯', icon: 'el-icon-search', roles: businessRoles }}
  ] },
  { path: '/evidence', component: Layout, children: [
    { path: 'index', name: 'Evidence', component: () => import('@/views/evidence/index'), meta: { title: '区块链存证查询', icon: 'el-icon-connection', roles: businessRoles }}
  ] },
  { path: '/admin/users', component: Layout, children: [
    { path: 'index', name: 'Users', component: () => import('@/views/admin/users'), meta: { title: '用户账号管理', icon: 'user', roles: ['系统管理员'] }}
  ] },
  { path: '/admin/logs', component: Layout, children: [
    { path: 'index', name: 'Logs', component: () => import('@/views/admin/logs'), meta: { title: '系统日志', icon: 'table', roles: ['系统管理员'] }}
  ] },
  { path: '*', redirect: '/404', hidden: true }
]
const createRouter = () => new Router({ scrollBehavior: () => ({ y: 0 }), routes: constantRoutes })
const router = createRouter()
export function resetRouter() { router.matcher = createRouter().matcher }
export default router
