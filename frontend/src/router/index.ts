import { createRouter, createWebHistory } from 'vue-router'
import InspectorView from '@/views/InspectorView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: InspectorView },
    { path: '/r/:id', name: 'detail', component: InspectorView, props: true },
  ],
})

export default router
