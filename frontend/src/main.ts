import { createApp } from 'vue'
import { ElButton, ElIcon, ElLoading, ElTag } from 'element-plus'
import 'element-plus/theme-chalk/base.css'
import 'element-plus/theme-chalk/el-button.css'
import 'element-plus/theme-chalk/el-icon.css'
import 'element-plus/theme-chalk/el-loading.css'
import 'element-plus/theme-chalk/el-message.css'
import 'element-plus/theme-chalk/el-tag.css'

import App from './App.vue'
import './styles/main.css'

const app = createApp(App)
const elementComponents = [ElButton, ElIcon, ElTag]

for (const component of elementComponents) {
  app.component(component.name!, component)
}

app.directive('loading', ElLoading.directive)
app.mount('#app')
