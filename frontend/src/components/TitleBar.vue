<script setup lang="ts">
import { WindowMinimise, WindowMaximise, WindowToggleMaximise, WindowUnmaximise, Quit } from '../../wailsjs/runtime/runtime'
import { ref, onMounted, onUnmounted } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import logoUrl from '../assets/images/logo-16.png'

const isMaximised = ref(false)

onMounted(() => {
  EventsOn('wails:window:resize', () => {
    // Check if maximised by comparing window size to screen
    isMaximised.value = window.outerWidth >= screen.availWidth - 2
  })
})

function minimize() {
  WindowMinimise()
}

function toggleMaximise() {
  if (isMaximised.value) {
    WindowUnmaximise()
  } else {
    WindowMaximise()
  }
  isMaximised.value = !isMaximised.value
}

function close() {
  Quit()
}
</script>

<template>
  <div class="title-bar">
    <div class="title-left">
      <img :src="logoUrl" class="app-logo" alt="Logo" />
      <span class="app-title">ApiClient</span>
    </div>
    <div class="title-drag-area" />
    <div class="title-controls">
      <button class="ctrl-btn" title="最小化" @click="minimize">
        <svg width="10" height="1" viewBox="0 0 10 1"><rect width="10" height="1" fill="currentColor" /></svg>
      </button>
      <button class="ctrl-btn" :title="isMaximised ? '还原' : '最大化'" @click="toggleMaximise">
        <svg v-if="isMaximised" width="10" height="10" viewBox="0 0 10 10">
          <rect x="2" y="0" width="8" height="8" fill="none" stroke="currentColor" stroke-width="1" />
          <rect x="0" y="2" width="8" height="8" fill="var(--app-bg)" stroke="currentColor" stroke-width="1" />
        </svg>
        <svg v-else width="10" height="10" viewBox="0 0 10 10">
          <rect x="0.5" y="0.5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1" />
        </svg>
      </button>
      <button class="ctrl-btn ctrl-close" title="关闭" @click="close">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <line x1="0" y1="0" x2="10" y2="10" stroke="currentColor" stroke-width="1.2" />
          <line x1="10" y1="0" x2="0" y2="10" stroke="currentColor" stroke-width="1.2" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.title-bar {
  display: flex;
  align-items: center;
  height: 32px;
  background-color: var(--app-bg-secondary);
  border-bottom: 1px solid var(--app-border);
  user-select: none;
  --wails-draggable: drag;
}

.title-left {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  flex-shrink: 0;
  height: 100%;
  --wails-draggable: drag;
}

.app-logo {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.app-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-accent);
  letter-spacing: 0.5px;
}

.title-drag-area {
  flex: 1;
  height: 100%;
  --wails-draggable: drag;
}

.title-controls {
  display: flex;
  height: 100%;
  --wails-draggable: no-drag;
}

.ctrl-btn {
  width: 46px;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--app-text-secondary);
  cursor: pointer;
  transition: background 0.1s, color 0.1s;
}

.ctrl-btn:hover {
  background-color: var(--app-bg-tertiary);
  color: var(--app-text);
}

.ctrl-close:hover {
  background-color: #e81123;
  color: #fff;
}
</style>
