import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from './stores/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('./views/LoginView.vue') },
    {
      path: '/',
      component: () => import('./views/ShellView.vue'),
      children: [
        { path: '', name: 'home', component: () => import('./views/EmptyView.vue') },
        { path: 'c/:id(\\d+)', name: 'chat', component: () => import('./views/ConversationView.vue'), props: (r) => ({ id: Number(r.params.id) }) },
        { path: 'admin', name: 'admin', component: () => import('./views/AdminView.vue') },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuth()
  if (!auth.checked) await auth.load()
  if (to.name === 'login') return auth.user ? { name: 'home' } : true
  if (!auth.user) return { name: 'login', query: { next: to.fullPath } }
  if (to.name === 'admin' && auth.user.role !== 'admin') return { name: 'home' }
  return true
})
