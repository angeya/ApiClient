<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { api } from '../../services/api'
import { useWorkspaceStore } from '../../stores/workspace'
import type { models } from '../../../wailsjs/go/models'

interface LoadTestProgress {
  id: string
  completed: number
  total: number
  percent: number
  rps: number
  downloadSpeed: number
  running: boolean
}

interface LoadTestResult {
  totalRequests: number
  durationSec: number
  successCount: number
  failCount: number
  errorRate: number
  totalBytesMB: number
  rps: number
  successRps: number
  bytesPerSecMB: number
  maxTimeMs: number
  minTimeMs: number
  avgTimeMs: number
}

const workspace = useWorkspaceStore()

const running = ref(false)
const loadTestId = ref('')
const progress = ref<LoadTestProgress | null>(null)
const result = ref<LoadTestResult | null>(null)

const config = ref<models.LoadTestConfig>({
  concurrency: 10,
  mode: 'rounds',
  rounds: 20,
  durationSeconds: 10,
  logEnabled: false,
})

const draft = computed(() => workspace.activeDraft)
const tab = computed(() => workspace.activeTab)

const expectedTotal = computed(() => {
  if (config.value.mode === 'rounds') {
    return config.value.concurrency * config.value.rounds
  }
  return 0
})

const rpsText = computed(() => {
  const val = result.value ? result.value.rps : (progress.value?.rps ?? 0)
  return val.toFixed(2) + ' r/s'
})

const speedText = computed(() => {
  const speed = progress.value?.downloadSpeed ?? 0
  if (speed >= 1024 * 1024) {
    return (speed / 1024 / 1024).toFixed(2) + ' m/s'
  }
  if (speed >= 1024) {
    return (speed / 1024).toFixed(2) + ' kb/s'
  }
  return speed.toFixed(2) + ' b/s'
})

function currentDirRel(rel: string): string {
  return rel.includes('\\') ? rel.substring(0, rel.lastIndexOf('\\')) : ''
}

function plainDraft(): any {
  if (!draft.value) return null
  return JSON.parse(JSON.stringify(draft.value))
}

async function start() {
  if (running.value) return
  if (!draft.value || !tab.value) {
    ElMessage.warning('请先选择一个请求')
    return
  }
  const req = plainDraft()
  if (!req) return

  result.value = null
  progress.value = null
  running.value = true
  loadTestId.value = ''

  try {
    const id = await api.startLoadTest(
      req,
      currentDirRel(tab.value.rel),
      config.value as models.LoadTestConfig
    )
    loadTestId.value = id
    ElMessage.success('压测已开始')
  } catch (err) {
    running.value = false
    ElMessage.error(`启动压测失败: ${err}`)
  }
}

async function stop() {
  if (!loadTestId.value) return
  try {
    await api.stopLoadTest(loadTestId.value)
    ElMessage.info('正在终止压测...')
  } catch (err) {
    ElMessage.error(`终止失败: ${err}`)
  }
}

function onProgress(data: LoadTestProgress) {
  if (data.id !== loadTestId.value) return
  progress.value = data
  if (!data.running) {
    running.value = false
  }
}

function onComplete(data: LoadTestResult) {
  result.value = data
  running.value = false
}

onMounted(() => {
  EventsOn('loadtest:progress', onProgress)
  EventsOn('loadtest:complete', onComplete)
})

onUnmounted(() => {
  EventsOff('loadtest:progress')
  EventsOff('loadtest:complete')
})
</script>

<template>
  <div class="load-test-tab">
    <div class="config-section">
      <el-form label-position="left" label-width="220px" class="load-form">
        <el-form-item label="并发数">
          <template #label>
            <span class="form-label">并发数</span>
            <span class="form-hint">（同时开启的工作进程数）</span>
          </template>
          <el-input-number v-model="config.concurrency" :min="1" :max="1000" :step="1" controls-position="right" />
        </el-form-item>

        <el-form-item label="模式">
          <el-radio-group v-model="config.mode">
            <el-radio label="rounds">按轮次</el-radio>
            <el-radio label="duration">按压测时长</el-radio>
          </el-radio-group>
          <template v-if="config.mode === 'rounds'">
            <span class="sub-label">压测轮次：</span>
            <el-input-number v-model="config.rounds" :min="1" :max="100000" :step="1" controls-position="right" />
          </template>
          <template v-else>
            <span class="sub-label">压测时长：</span>
            <el-input-number v-model="config.durationSeconds" :min="1" :max="3600" :step="1" controls-position="right" />
            <span class="unit">秒</span>
          </template>
        </el-form-item>

        <el-form-item>
          <template #label>
            <span class="form-label">开启日志</span>
            <span class="form-hint">（开启日志对性能有影响）</span>
          </template>
          <el-switch v-model="config.logEnabled" />
        </el-form-item>
      </el-form>

      <div class="action-bar">
        <el-button v-if="!running" type="primary" size="default" @click="start">开始压测</el-button>
        <el-button v-else type="danger" size="default" @click="stop">终止压测</el-button>
        <span v-if="expectedTotal > 0" class="expected-total">预计总请求数：{{ expectedTotal }}</span>
      </div>
    </div>

    <div class="result-section">
      <div class="section-title">压测结果</div>

      <div class="metric-cards">
        <div class="metric-card">
          <div class="metric-label">{{ result ? '每秒请求数（平均）' : '每秒请求数' }}</div>
          <div class="metric-value">{{ rpsText }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">下载速度</div>
          <div class="metric-value speed">{{ speedText }}</div>
        </div>
      </div>

      <div v-if="running && progress" class="progress-area">
        <el-progress :percentage="Math.round(progress.percent)" :stroke-width="12" status="success" />
        <div class="progress-text">{{ Math.round(progress.percent) }}%</div>
      </div>

      <template v-if="result">
        <div class="section-subtitle">总结果</div>
        <table class="result-table">
          <thead>
            <tr>
              <th>总请求数</th>
              <th>执行时间（s）</th>
              <th>成功请求数</th>
              <th>失败请求数</th>
              <th>错误率</th>
              <th>总接收数据（M）</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>{{ result.totalRequests }}</td>
              <td>{{ result.durationSec.toFixed(2) }}</td>
              <td>{{ result.successCount }}</td>
              <td>{{ result.failCount }}</td>
              <td>{{ result.errorRate.toFixed(2) }}%</td>
              <td>{{ result.totalBytesMB.toFixed(2) }}</td>
            </tr>
          </tbody>
        </table>

        <div class="section-subtitle">主要指标</div>
        <table class="result-table">
          <thead>
            <tr>
              <th>每秒请求数</th>
              <th>每秒成功请求数</th>
              <th>每秒接收字节数（M）</th>
              <th>最大响应时间（ms）</th>
              <th>最小响应时间（ms）</th>
              <th>平均响应时间（ms）</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>{{ result.rps.toFixed(2) }}</td>
              <td>{{ result.successRps.toFixed(2) }}</td>
              <td>{{ result.bytesPerSecMB.toFixed(2) }}</td>
              <td>{{ result.maxTimeMs }}</td>
              <td>{{ result.minTimeMs }}</td>
              <td>{{ result.avgTimeMs.toFixed(2) }}</td>
            </tr>
          </tbody>
        </table>
      </template>
    </div>
  </div>
</template>

<style scoped>
.load-test-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  overflow: auto;
  gap: 24px;
}

.config-section {
  flex-shrink: 0;
}

.load-form {
  max-width: 700px;
}

.form-label {
  color: var(--app-text);
}

.form-hint {
  color: var(--app-text-secondary);
  font-size: 12px;
}

.sub-label {
  margin-left: 16px;
  margin-right: 8px;
  color: var(--app-text);
  font-size: 13px;
}

.unit {
  margin-left: 6px;
  color: var(--app-text-secondary);
  font-size: 13px;
}

.action-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
}

.expected-total {
  font-size: 13px;
  color: var(--app-text-secondary);
}

.result-section {
  flex: 1;
  min-height: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-accent);
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--app-border);
}

.section-subtitle {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  margin: 20px 0 10px;
}

.progress-area {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.progress-area :deep(.el-progress) {
  flex: 1;
}

.progress-text {
  font-size: 13px;
  color: var(--app-text-secondary);
  min-width: 40px;
  text-align: right;
}

.metric-cards {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.metric-card {
  width: 180px;
  padding: 16px;
  border-radius: 6px;
  background-color: var(--app-bg-secondary);
  border: 1px solid var(--app-border);
}

.metric-label {
  font-size: 12px;
  color: var(--app-text-secondary);
  margin-bottom: 8px;
}

.metric-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--app-danger, #f56c6c);
  font-family: 'Consolas', 'Monaco', monospace;
}

.metric-value.speed {
  color: var(--app-success, #67c23a);
}

.result-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.result-table th,
.result-table td {
  padding: 10px 12px;
  text-align: left;
  border: 1px solid var(--app-border);
}

.result-table th {
  background-color: var(--app-bg-secondary);
  color: var(--app-text-secondary);
  font-weight: 600;
}

.result-table td {
  color: var(--app-text);
}
</style>
