import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/RegisterView.vue'),
    },
    {
      path: '/',
      name: 'gallery',
      component: () => import('../views/GalleryView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/timeline',
      name: 'timeline',
      component: () => import('../views/TimelineView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/map',
      name: 'map',
      component: () => import('../views/MapView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/folders',
      name: 'folders',
      component: () => import('../views/FoldersView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/AdminView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next('/login')
  } else if (to.meta.requiresAdmin && auth.user?.role !== 'admin') {
    next('/')
  } else if ((to.path === '/login' || to.path === '/register') && auth.isAuthenticated) {
    next('/')
  } else {
    next()
  }
})

export default router
