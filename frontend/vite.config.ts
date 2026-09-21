import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    // 按需引入 Element Plus 的组件与样式：
    // 全量注册会引入所有组件及 390KB 的 CSS，明显拖慢首屏，改为按模板实际使用自动引入
    Components({
      resolvers: [ElementPlusResolver({ importStyle: 'css' })],
      dts: false,
    }),
  ],
  build: {
    target: 'esnext',
    sourcemap: false,
  },
})
