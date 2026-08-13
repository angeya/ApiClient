import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Theme = 'dark-1' | 'dark-2' | 'warm'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>('dark-1')

  function setTheme(value: Theme) {
    theme.value = value
    document.documentElement.dataset.theme = value
  }

  function initTheme(saved?: string) {
    const valid: Theme[] = ['dark-1', 'dark-2', 'warm']
    const t = valid.includes(saved as Theme) ? (saved as Theme) : 'dark-1'
    setTheme(t)
  }

  watch(theme, (v) => {
    document.documentElement.dataset.theme = v
  })

  return { theme, setTheme, initTheme }
})
