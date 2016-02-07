import Vue from 'vue'
import App from './components/App.vue'
import Home from './components/Home.vue'
import Dashboard from './components/Dashboard.vue'
import Settings from './components/Settings.vue'
import VueRouter from 'vue-router'
import VueResource from 'vue-resource'
Vue.use(VueResource)
Vue.use(VueRouter)

var router = new VueRouter()

router.map({
  '/': {
    component: Home
  },
  '/dashboard': {
    component: Dashboard
  },
  '/settings': {
    component: Settings
  }
})

router.redirect({
  '*': '/'
})

Vue.http.interceptors.push({
  response: function (response) {
    if (response.status === 401) {
      this.logout();
      this.authenticated = false;      
      router.go('/');
    }
    return response;
  }
})

router.start(App, '#app')
