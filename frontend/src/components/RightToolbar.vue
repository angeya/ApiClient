<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkspaceStore } from '../stores/workspace'
import VariablesDrawer from './VariablesDrawer.vue'
import SettingsDialog from './SettingsDialog.vue'
import SwaggerImportDialog from './SwaggerImportDialog.vue'

const workspace = useWorkspaceStore()
const appVersion = 'v0.0.1'

const varsDrawerVisible = ref(false)
const settingsVisible = ref(false)
const importVisible = ref(false)

function openImport() {
  if (!workspace.root) {
    ElMessage.warning('请先打开工作空间')
    return
  }
  importVisible.value = true
}
</script>

<template>
  <div class="right-toolbar">
    <div class="toolbar-actions">
      <button class="tool-btn" title="设置" @click="settingsVisible = true">
        <el-icon :size="18"><Tools /></el-icon>
      </button>
      <button class="tool-btn" title="全局参数" @click="varsDrawerVisible = true">
        <el-icon :size="18"><Collection /></el-icon>
      </button>
      <button class="tool-btn" title="导入接口" @click="openImport">
        <el-icon :size="18"><Upload /></el-icon>
      </button>
    </div>

    <div class="toolbar-footer">
      <span class="version-text">目前版本：{{ appVersion }}</span>
    </div>

    <VariablesDrawer v-model="varsDrawerVisible" />
    <SettingsDialog v-model="settingsVisible" />
    <SwaggerImportDialog v-model="importVisible" />
  </div>
</template>

<style scoped>
.right-toolbar {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 8px;
  height: 100%;
}

.toolbar-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding-top: 8px;
  flex: 1;
}

.toolbar-footer {
  padding: 8px 0 12px;
}

.version-text {
  font-size: 10px;
  color: var(--app-text-secondary);
  writing-mode: vertical-rl;
  letter-spacing: 1px;
  white-space: nowrap;
}

.tool-btn {
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

.tool-btn:hover {
  background-color: var(--app-bg-tertiary);
  color: var(--app-accent);
}

.tool-btn:active {
  transform: scale(0.92);
}
</style>
