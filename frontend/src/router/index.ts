import { createRouter, createWebHistory, type Router } from 'vue-router'
import InspectorView from '@/views/InspectorView.vue'
import LoginView from '@/views/LoginView.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: InspectorView },
    { path: '/r/:id', name: 'detail', component: InspectorView, props: true },
    { path: '/login', name: 'login', component: LoginView },
  ],
})

// setupGuard 安装登录守卫；抽成具名导出便于单测用 memory router 复用同一套逻辑。
export function setupGuard(r: Router) {
  r.beforeEach(async (to) => {
    const auth = useAuthStore()
    if (!auth.ready) await auth.fetchSession()
    if (auth.enabled && !auth.authenticated && to.name !== 'login') {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
    if (to.name === 'login' && (!auth.enabled || auth.authenticated)) {
      return { name: 'home' }
    }
    return true
  })
}

setupGuard(router)

export default router
