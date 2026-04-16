import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/main.css'

// ensure dark mode on load (html already has .dark class for SSR default)
if (!document.documentElement.classList.contains('dark')) {
  document.documentElement.classList.add('dark')
}

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
