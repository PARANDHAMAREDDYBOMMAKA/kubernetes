import {
  createRouter,
  createWebHistory,
  type RouteRecordRaw
} from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/Login.vue'),
    meta: { public: true, layout: 'auth' }
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/pages/Register.vue'),
    meta: { public: true, layout: 'auth' }
  },
  {
    path: '/',
    component: () => import('@/layouts/AppLayout.vue'),
    redirect: { name: 'clusters' },
    children: [
      {
        path: 'clusters',
        name: 'clusters',
        component: () => import('@/pages/Clusters.vue'),
        meta: { title: 'Clusters' }
      },
      {
        path: 'clusters/:id',
        name: 'cluster-detail',
        component: () => import('@/pages/ClusterDetail.vue'),
        meta: { title: 'Cluster' },
        props: true
      },
      {
        path: 'loadbalancers',
        name: 'loadbalancers',
        component: () => import('@/pages/LoadBalancers.vue'),
        meta: { title: 'Load Balancers' }
      },
      {
        path: 'loadbalancers/:id',
        name: 'loadbalancer-detail',
        component: () => import('@/pages/LoadBalancerDetail.vue'),
        meta: { title: 'Load Balancer' },
        props: true
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/pages/Settings.vue'),
        meta: { title: 'Settings' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFound.vue'),
    meta: { public: true, layout: 'auth' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  }
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  const isPublic = to.meta.public === true
  if (!isPublic && !auth.token) {
    return {
      name: 'login',
      query: to.fullPath && to.fullPath !== '/' ? { redirect: to.fullPath } : {}
    }
  }
  if (auth.token && (to.name === 'login' || to.name === 'register')) {
    return { name: 'clusters' }
  }
  return true
})

export default router
