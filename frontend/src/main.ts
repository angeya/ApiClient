import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { icons } from './icons'
// ElMessage / ElMessageBox / ElNotification 以函数形式调用，插件无法从模板推断，需显式补样式
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/notification/style/css'
import './styles/global.css'
import './styles/themes/dark-1.css'
import './styles/themes/dark-2.css'
import './styles/themes/warm.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
for (const [key, component] of Object.entries(icons)) {
  app.component(key, component)
}
app.mount('#app')
