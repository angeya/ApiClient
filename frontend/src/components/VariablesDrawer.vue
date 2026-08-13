<script setup lang="ts">
import { computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkspaceStore } from '../stores/workspace'
import KeyValueEditor from './common/KeyValueEditor.vue'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const workspace = useWorkspaceStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

async function save() {
  await workspace.saveSettings()
  ElMessage.success('全局参数已保存')
}

watch(
  () => visible.value,
  async (val) => {
    if (val) {
      await workspace.loadSettings()
    }
  }
)
</script>

<template>
  <el-drawer v-model="visible" title="全局变量与请求头" size="40%">
    <el-tabs type="border-card">
      <el-tab-pane label="全局变量">
        <KeyValueEditor v-model="workspace.settings.globalVariables" value-label="变量值" />
      </el-tab-pane>
      <el-tab-pane label="全局请求头">
        <KeyValueEditor v-model="workspace.settings.globalHeaders" />
      </el-tab-pane>
    </el-tabs>
    <div class="drawer-actions">
      <el-button type="primary" @click="save">保存</el-button>
    </div>
  </el-drawer>
</template>

<style scoped>
.drawer-actions {
  margin-top: 16px;
}
</style>
