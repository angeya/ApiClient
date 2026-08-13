<script setup lang="ts">
import { ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useThemeStore, type Theme } from '../stores/theme'

const workspace = useWorkspaceStore()
const theme = useThemeStore()
const showPicker = ref(false)

async function selectTheme(t: Theme) {
  theme.setTheme(t)
  workspace.settings.theme = t
  await workspace.saveSettings()
  showPicker.value = false
}
</script>

<template>
  <el-popover :visible="showPicker" placement="left" :width="120" @update:visible="showPicker = $event">
    <template #reference>
      <button class="theme-toggle-btn" title="切换主题" @click="showPicker = !showPicker">
        <el-icon :size="18"><Brush /></el-icon>
      </button>
    </template>
    <div class="theme-list">
      <div class="theme-item" @click="selectTheme('dark-1')">深色主题 1</div>
      <div class="theme-item" @click="selectTheme('dark-2')">深色主题 2</div>
      <div class="theme-item" @click="selectTheme('warm')">暖色主题</div>
    </div>
  </el-popover>
</template>

<style scoped>
.theme-toggle-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.theme-toggle-btn:hover {
  background-color: var(--app-bg-tertiary);
  color: var(--app-accent);
}

.theme-toggle-btn:active {
  transform: scale(0.92);
}

.theme-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.theme-item {
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.1s;
}

.theme-item:hover {
  background-color: var(--app-bg-tertiary);
}
</style>
