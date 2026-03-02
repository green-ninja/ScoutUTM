import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import LoginView from '@/views/LoginView.vue'
import DashboardView from '@/views/DashboardView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/wizard',
      name: 'wizard',
      component: () => import('@/views/SetupWizardView.vue'),
    },
    {
      path: '/',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: DashboardView,
        },
        {
          path: 'firewall',
          name: 'firewall',
          component: () => import('@/views/FirewallView.vue'),
        },
        {
          path: 'dns',
          name: 'dns',
          component: () => import('@/views/DnsView.vue'),
        },
        {
          path: 'vpn',
          name: 'vpn',
          component: () => import('@/views/VpnView.vue'),
        },
        {
          path: 'devices',
          name: 'devices',
          component: () => import('@/views/DevicesView.vue'),
        },
        {
          path: 'ids',
          name: 'ids',
          component: () => import('@/views/IDSView.vue'),
        },
        {
          path: 'parental',
          name: 'parental',
          component: () => import('@/views/ParentalView.vue'),
        },
        {
          path: 'network',
          name: 'network',
          component: () => import('@/views/NetworkView.vue'),
        },
        {
          path: 'system',
          name: 'system',
          component: () => import('@/views/SystemView.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard',
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  // Wizard route: accessible without auth (uses its own auth for API calls).
  if (to.name === 'wizard') return

  if (to.name !== 'login' && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router
