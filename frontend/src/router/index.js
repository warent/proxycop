import Vue from 'vue'
import Router from 'vue-router'
import Home from '@/components/Home'
import URLStatus from '@/components/URLStatus'

Vue.use(Router)

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'Home',
      component: Home
    },
    {
      path: '/url/:url/status',
      name: 'URLStatus',
      component: URLStatus
    }
  ]
})
