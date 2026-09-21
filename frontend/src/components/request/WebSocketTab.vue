<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { useWorkspaceStore } from '../../stores/workspace'
import { api } from '../../services/api'

interface WsMessage {
  text: string
  type: 'in' | 'out' | 'error'
  time: string
}

const props = defineProps<{
  connected: boolean
}>()

const emit = defineEmits<{
  (e: 'disconnected'): void
}>()

const workspace = useWorkspaceStore()

const messages = ref<WsMessage[]>([])
const messageInput = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const draft = computed(() => workspace.activeDraft)
const requestId = computed(() => draft.value?.id || '')

function formatTime(date = new Date()): string {
  const h = date.getHours().toString().padStart(2, '0')
  const m = date.getMinutes().toString().padStart(2, '0')
  const s = date.getSeconds().toString().padStart(2, '0')
  return `${h}:${m}:${s}`
}

function appendMessage(text: string, type: 'in' | 'out' | 'error') {
  messages.value.push({
    text,
    type,
    time: formatTime(),
  })
  nextTick(() => {
    const el = messageListRef.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}

function clearMessages() {
  messages.value = []
}

async function sendMessage() {
  const req = draft.value
  if (!req || !props.connected) return
  const text = messageInput.value
  if (!text.trim()) return
  try {
    await api.sendWebSocketMessage(req.id, text)
    appendMessage(text, 'out')
    messageInput.value = ''
  } catch (err) {
    ElMessage.error(`发送失败: ${err}`)
  }
}

function onMessage(id: string, data: string) {
  if (id !== requestId.value) return
  appendMessage(data, 'in')
}

function onError(id: string, data: string) {
  if (id !== requestId.value) return
  appendMessage(data, 'error')
  emit('disconnected')
}

watch(requestId, () => {
  // 切换请求时清空消息，避免不同连接的消息混淆
  clearMessages()
})

onMounted(() => {
  EventsOn('websocket:message', onMessage)
  EventsOn('websocket:error', onError)
})

onUnmounted(() => {
  EventsOff('websocket:message')
  EventsOff('websocket:error')
})
</script>

<template>
  <div class="websocket-tab">
    <div class="ws-status-bar">
      <span :class="['status-dot', connected ? 'connected' : 'disconnected']" />
      <span class="status-text">{{ connected ? '已连接' : '未连接' }}</span>
      <span v-if="messages.length" class="message-count">{{ messages.length }} 条消息</span>
      <div class="spacer" />
      <el-button v-if="messages.length" size="small" text @click="clearMessages">清空消息</el-button>
    </div>

    <div ref="messageListRef" class="ws-message-list">
      <div
        v-for="(msg, idx) in messages"
        :key="idx"
        :class="['ws-message', msg.type]"
      >
        <div class="ws-message-header">
          <span class="ws-badge">{{ msg.type === 'in' ? '← 接收' : msg.type === 'out' ? '→ 发送' : '! 错误' }}</span>
          <span class="ws-time">{{ msg.time }}</span>
        </div>
        <pre class="ws-message-body">{{ msg.text }}</pre>
      </div>
      <el-empty v-if="!messages.length" description="连接后即可收发消息" :image-size="60" />
    </div>

    <div class="ws-input-area">
      <el-input
        v-model="messageInput"
        type="textarea"
        :rows="3"
        resize="none"
        placeholder="输入要发送的消息，Ctrl + Enter 发送"
        :disabled="!connected"
        @keydown.ctrl.enter.prevent="sendMessage"
      />
      <el-button
        type="primary"
        :disabled="!connected || !messageInput.trim()"
        @click="sendMessage"
      >
        发送
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.websocket-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 12px;
  gap: 12px;
}

.ws-status-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 4px;
  background-color: var(--app-bg-secondary);
  border: 1px solid var(--app-border);
  font-size: 13px;
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--app-text-secondary);
}

.status-dot.connected {
  background-color: var(--app-success);
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.status-dot.disconnected {
  background-color: var(--app-text-secondary);
}

.status-text {
  color: var(--app-text);
  font-weight: 600;
}

.message-count {
  color: var(--app-text-secondary);
  font-size: 12px;
}

.spacer {
  flex: 1;
}

.ws-message-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  border-radius: 4px;
  background-color: var(--app-bg-secondary);
  border: 1px solid var(--app-border);
}

.ws-message {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 4px;
  background-color: var(--app-bg);
  border: 1px solid var(--app-border);
  font-size: 12px;
}

.ws-message.out {
  border-left: 3px solid var(--app-accent);
}

.ws-message.in {
  border-left: 3px solid var(--app-success);
}

.ws-message.error {
  border-left: 3px solid var(--app-danger);
  background-color: rgba(244, 67, 54, 0.08);
}

.ws-message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.ws-badge {
  font-weight: 600;
  color: var(--app-text-secondary);
}

.ws-time {
  color: var(--app-text-secondary);
  font-size: 11px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.ws-message-body {
  margin: 0;
  padding: 0;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--app-text);
  line-height: 1.5;
}

.ws-input-area {
  display: flex;
  gap: 10px;
  align-items: flex-end;
  flex-shrink: 0;
}

.ws-input-area :deep(.el-textarea__inner) {
  background-color: var(--app-bg) !important;
  border-color: var(--app-border) !important;
  color: var(--app-text);
}

.ws-input-area :deep(.el-textarea__inner:focus) {
  border-color: var(--app-accent) !important;
}

.ws-input-area .el-button {
  flex-shrink: 0;
  height: 74px;
  padding: 0 24px;
}
</style>
