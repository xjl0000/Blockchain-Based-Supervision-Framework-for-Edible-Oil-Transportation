import router from './router'
import store from './store'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'
import { getToken } from '@/utils/auth'

NProgress.configure({ showSpinner: false })
router.beforeEach(async(to, from, next) => {
  NProgress.start()
  if (!getToken()) {
    next(to.path === '/login' ? undefined : `/login?redirect=${to.path}`)
    NProgress.done()
    return
  }
  if (to.path === '/login') { next('/'); NProgress.done(); return }
  if (!store.getters.role) {
    try { await store.dispatch('user/getInfo') } catch (e) { await store.dispatch('user/resetToken'); next('/login'); NProgress.done(); return }
  }
  const roles = to.meta && to.meta.roles
  next(!roles || roles.indexOf(store.getters.role) >= 0 ? undefined : `/forbidden?from=${encodeURIComponent(to.path)}`)
})
router.afterEach(() => NProgress.done())
