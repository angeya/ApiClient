# 高性能轻量级 API 客户端工具设计文档

## 1. 项目概述

开发一款基于 Wails 2 + Vue 3 + Element Plus + TypeScript 的桌面 API 客户端工具，聚焦核心接口调试能力，支持 HTTP/HTTPS 与 WebSocket 协议。目标是：**无需登录、低内存占用（≤100MB）、启动快速（≤3秒）、界面响应迅速（≤100ms）**。

产品定位为开发者本地/团队共享的接口调试器，数据以 JSON 文件形式持久化到本地工作区，天然支持 Git 版本管理。

---

## 2. 系统架构设计

### 2.1 总体架构

采用 **Wails 2 经典桌面应用架构**：

```
┌─────────────────────────────────────────────────────────────┐
│                        前端 (Frontend)                       │
│   Vue 3 + Element Plus + Pinia + Vue Router + Monaco Editor │
│   负责 UI 渲染、状态管理、脚本编辑、主题切换                  │
├─────────────────────────────────────────────────────────────┤
│                    Wails Runtime Bridge                      │
│   Events / Bindings（Go ↔ JS 双向绑定）                       │
├─────────────────────────────────────────────────────────────┤
│                        后端 (Backend)                        │
│   Go + Wails：业务编排、文件 IO、HTTP/WebSocket 执行、剪贴板  │
│   监控、脚本沙箱（QuickJS/goja）、环境变量解析                │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 模块划分

| 模块 | 职责 | 主要位置 |
|------|------|----------|
| Workspace | 工作区管理：目录/接口/环境/全局变量的增删改查与 JSON 文件持久化 | Go: `internal/workspace`, TS: `src/stores/workspace.ts` |
| Request | HTTP/WebSocket 请求构建、发送、取消、响应解析 | Go: `internal/request`, TS: `src/services/request.ts` |
| Headers | 三级请求头继承计算（全局→目录→接口） | Go: `internal/headers`, TS: `src/utils/headers.ts` |
| Variables | 全局/环境变量定义、引用、动态替换 | Go: `internal/variables`, TS: `src/utils/variables.ts` |
| Environments | 多环境配置：名称、基础 URL、环境变量 | Go: `internal/environment`, TS: `src/stores/environment.ts` |
| Hooks | 前置/后置 JS 脚本执行，提供 `pm.*` 风格 API | Go: `internal/hooks`, TS: `src/components/HookEditor.vue` |
| Theme | 三套主题切换与持久化 | TS: `src/styles/themes/`, `src/stores/theme.ts` |
| Fetch Import | 剪贴板监听、Fetch API 代码解析、填充表单 | Go: `internal/fetch`, TS: `src/utils/fetchParser.ts` |
| UI Layout | 主界面布局、侧边栏、请求编辑器、响应面板 | TS: `src/views/`, `src/components/` |
| Persistence | 保存状态跟踪、关闭前未保存检查、导入导出 | Go: `internal/persistence`, TS: `src/stores/dirty.ts` |

### 2.3 进程与线程模型

- 主线程：Wails 主窗口 + Go 后端服务。
- 请求执行：每个 HTTP/WebSocket 请求在独立 Go goroutine 中运行，支持取消。
- 脚本执行：使用 QuickJS（CGO）或 goja（纯 Go）沙箱执行前后置脚本，与主 goroutine 隔离，设置超时防止死循环。
- 剪贴板监听：由 Go 后端启动后台 goroutine，间隔 1 秒轮询剪贴板文本，检测到 Fetch 代码时通过 Wails Events 通知前端。

---

## 3. 数据模型设计

### 3.1 工作区目录结构

工作区是一个普通文件夹，内部按以下结构组织：

```
my-api-workspace/
├── .apiclient/
│   ├── settings.json          # 全局设置：主题、最近环境、全局请求头、全局变量
│   └── environments/
│       ├── dev.json
│       ├── prod.json
│       └── test.json
├── users/
│   ├── _dir.meta.json         # 目录元数据：目录级请求头
│   ├── login.json             # 接口文件
│   └── profile.json
└── orders/
    ├── _dir.meta.json
    └── create.json
```

命名约定：
- 接口/目录文件均使用小写、短横线命名，避免空格与特殊字符。
- 目录元数据文件名为 `_dir.meta.json`。
- 全局配置目录 `.apiclient` 为隐藏目录。

### 3.2 JSON Schema

#### 3.2.1 全局设置 `settings.json`

```json
{
  "version": "1.0.0",
  "theme": "dark-1",
  "activeEnvironmentId": "dev",
  "globalHeaders": [
    { "key": "User-Agent", "value": "ApiClient/1.0", "enabled": true, "description": "" }
  ],
  "globalVariables": [
    { "key": "token", "value": "{{env.token}}", "description": "登录令牌" }
  ]
}
```

#### 3.2.2 目录元数据 `_dir.meta.json`

```json
{
  "id": "dir-users",
  "name": "users",
  "headers": [
    { "key": "Accept", "value": "application/json", "enabled": true }
  ],
  "variables": []
}
```

#### 3.2.3 接口文件 `*.json`

```json
{
  "id": "req-login",
  "type": "http",
  "name": "用户登录",
  "method": "POST",
  "url": "{{baseUrl}}/api/auth/login",
  "headers": [
    { "key": "Content-Type", "value": "application/json", "enabled": true }
  ],
  "query": [
    { "key": "version", "value": "v1", "enabled": true }
  ],
  "bodyType": "json",
  "body": "{\"username\":\"admin\",\"password\":\"123456\"}",
  "formData": [],
  "preRequestScript": "pm.log('sending login request');",
  "postResponseScript": "pm.setGlobal('token', pm.response.json().token);",
  "description": "登录接口"
}
```

字段说明：
- `type`: `http` | `websocket`
- `bodyType`: `none` | `json` | `text` | `form-data` | `x-www-form-urlencoded`
- `formData`: 数组，元素含 `key`, `value`, `type` (`text`/`file`), `enabled`
- `query`: URL 查询参数
- `preRequestScript` / `postResponseScript`: JS 脚本字符串

#### 3.2.4 环境文件 `environments/*.json`

```json
{
  "id": "env-dev",
  "name": "开发环境",
  "baseUrl": "http://localhost:8080",
  "variables": [
    { "key": "token", "value": "dev-token", "description": "" }
  ]
}
```

### 3.3 运行时内存模型

前端使用 Pinia 维护一棵**目录/接口树**和当前激活请求的**运行时副本**：

```
workspaceStore:
  - tree: ApiNode[]
  - activeRequestId: string
  - draft: RequestDraft   // 当前编辑态，未保存
  - dirty: Map<string, boolean>
```

编辑仅修改 `draft`，点击“保存”后才写回 JSON 文件并同步 `tree`。

---

## 4. UI/UX 设计规范

### 4.1 布局

采用经典三栏布局：

```
┌────────────┬──────────────────────────────────────┬───────────────┐
│  侧边栏     │           请求编辑区                    │    响应面板    │
│  - 目录树   │  - 方法/URL/发送                       │  - 状态/耗时   │
│  - 环境切换 │  - Params / Headers / Body / Scripts  │  - Body /     │
│  - 全局设置 │  - Pre-request / Post-response        │    Headers /  │
│            │                                       │    Cookies    │
└────────────┴──────────────────────────────────────┴───────────────┘
```

尺寸：
- 侧边栏宽度：260px，可折叠。
- 响应区宽度：占右侧 45%，可拖动调整。
- 垂直间距：Element Plus 默认间距。

### 4.2 主题

使用 CSS 变量定义三套主题，切换时仅修改根元素 `data-theme`：

| 主题 | 主背景 | 次背景 | 主文字 | 强调色 | 用途 |
|------|--------|--------|--------|--------|------|
| dark-1 | #1e1e1e | #252526 | #d4d4d4 | #4ec9b0 | 夜间 |
| dark-2 | #0d1117 | #161b22 | #c9d1d9 | #58a6ff | 深色替代 |
| warm | #f5f0e8 | #efe9dd | #3e3a36 | #d97706 | 低蓝光 |

Element Plus 主题通过 `element-plus/theme-chalk` 自定义 + CSS 变量覆盖。

### 4.3 交互规范

- 未保存标签显示圆点或修改标识。
- 关闭窗口/标签前若 `dirty` 为 true，弹出确认对话框。
- 请求头发送前展示“最终合并结果”预览。
- 变量引用使用 `{{varName}}`，悬停显示解析值。
- 错误提示使用 Element Plus `ElMessage`，不阻断操作。

---

## 5. 关键功能实现方案

### 5.1 三级请求头继承

合并顺序：全局 → 目录 → 接口。重复 key 时后者覆盖前者，但可通过 `enabled=false` 禁用某一级别的特定 header。

算法伪代码：

```typescript
function mergeHeaders(global: Header[], dir: Header[], req: Header[]): Header[] {
  const map = new Map<string, Header>();
  [...global, ...dir, ...req]
    .filter(h => h.enabled)
    .forEach(h => map.set(h.key.toLowerCase(), h));
  return Array.from(map.values());
}
```

### 5.2 变量解析

变量作用域优先级：**接口级变量 > 目录级变量 > 环境变量 > 全局变量**。

解析器支持嵌套变量与默认值：
- `{{baseUrl}}`
- `{{token | default}}`
- `{{$timestamp}}` 内置动态变量

实现位置：Go 后端提供 `Variables.Resolve(input, scope)`，前端同步实现预览解析。

### 5.3 请求发送

HTTP 请求流程：
1. 前端组装请求草稿。
2. 调用 Go `SendHttpRequest(request)`。
3. Go 端执行前置脚本（修改 request）。
4. 解析变量，合并请求头，构造 `http.Request`。
5. 发送请求，记录耗时，支持取消。
6. 执行后置脚本（读取 response，设置变量）。
7. 返回结构化响应给前端。

WebSocket 请求流程：
1. 前端调用 `ConnectWebSocket(request)`。
2. Go 使用 `gorilla/websocket` 建立连接。
3. 通过 Wails Events 持续推送接收到的消息。
4. 提供 `SendWebSocketMessage` / `DisconnectWebSocket`。

### 5.4 前后置脚本

脚本运行在一个受限 JS 沙箱中，暴露 `pm` 全局对象：

```javascript
// 前置脚本
pm.log('info', 'start');
pm.setEnv('requestId', '123');
pm.request.url = 'https://api.example.com/users';
pm.request.headers.add('X-Trace', 'trace-001');

// 后置脚本
pm.log('info', pm.response.status);
pm.setGlobal('token', pm.response.json().data.token);
```

`pm` API 设计：
- `pm.log(level, ...args)`
- `pm.setGlobal(key, value)` / `pm.getGlobal(key)`
- `pm.setEnv(key, value)` / `pm.getEnv(key)`
- `pm.request.url`, `pm.request.method`, `pm.request.headers`, `pm.request.body`
- `pm.response.status`, `pm.response.headers`, `pm.response.text()`, `pm.response.json()`

沙箱选型：**goja**（纯 Go，无需 CGO，体积小，性能满足需求）。

### 5.5 环境管理

- 环境文件存放在 `.apiclient/environments/`。
- 切换环境时更新 `settings.json` 中的 `activeEnvironmentId`，前端重新解析所有变量。
- 导入导出：单个环境 JSON 文件导入/导出；全部环境打包为 zip。

### 5.6 主题系统

- 主题名写入 `settings.json`。
- 应用启动时读取并设置 `document.documentElement.dataset.theme`。
- Element Plus 通过 SCSS 变量动态覆盖，CSS 自定义属性实现即时切换。

### 5.7 Fetch API 导入

- 后端启动剪贴板监听 goroutine，检测到包含 `fetch(` 的文本时通知前端。
- 前端或后端解析 Fetch 调用：提取 URL、method、headers、body、mode 等。
- 自动生成新的接口草稿并打开编辑。
- 同时提供手动导入弹窗。

解析器覆盖常见写法：
- `fetch(url, { method, headers, body })`
- `await fetch(...)`
- 使用变量或字符串字面量

### 5.8 保存与脏状态

- 每个接口草稿维护 `dirty` 标记。
- 快捷键 `Ctrl+S` 触发保存。
- 保存时：
  1. 校验 JSON body 合法性（bodyType=json 时）。
  2. 将草稿写入对应 JSON 文件。
  3. 更新 tree 节点元数据。
  4. 清除 dirty。
- 关闭应用前：后端提供 `BeforeClose` 事件，前端检查 dirty，未保存则弹出确认框。

### 5.9 导入导出

- 导入：选择 `.json` 接口文件或包含多个接口的文件夹，合并到当前工作区。
- 导出：单个接口导出为 JSON；选中目录导出为 zip；整个工作区导出为 zip。

---

## 6. 性能设计

### 6.1 启动优化

- 工作区树采用**懒加载**：仅加载目录结构和文件元数据，不预读所有接口文件内容。
- 首次打开时只读取 `settings.json` 和环境文件。
- 使用 Vite 构建前端，启用代码分割与 Tree Shaking。
- Go 后端编译为静态二进制，减少启动开销。

### 6.2 内存控制

- 响应体默认限制 10MB，超大响应提供“保存到文件”选项。
- 关闭的接口标签释放运行时副本，仅保留树节点引用。
- WebSocket 消息缓冲区限制 1000 条，超出时滚动移除旧消息。
- 不使用本地数据库引擎，避免常驻内存索引。

### 6.3 响应速度

- 前端使用虚拟滚动展示大型响应体。
- 请求头发送预览延迟计算，防抖 200ms。
- 文件写入使用批量/异步方式，保存按钮触发同步写入但界面不阻塞。

---

## 7. 开发计划与里程碑

### 阶段一：项目骨架（第 1 天）

- [ ] 初始化 Wails 2 + Vue 3 + TypeScript 项目
- [ ] 配置 Element Plus、Pinia、Vue Router
- [ ] 配置代码规范（ESLint + Prettier）
- [ ] 搭建主界面布局与主题系统

### 阶段二：工作区与数据层（第 2 天）

- [ ] 实现 JSON 文件工作区读写（目录树、接口文件、环境文件、settings）
- [ ] 实现 Pinia workspace store 与 dirty 跟踪
- [ ] 实现导入导出基础能力

### 阶段三：请求核心（第 3 天）

- [ ] 实现 HTTP/HTTPS 请求执行与取消
- [ ] 实现 WebSocket 连接、收发、断开
- [ ] 实现变量解析与请求头合并

### 阶段四：编辑器与参数（第 4 天）

- [ ] 实现请求编辑器（方法、URL、Query、Headers、Body）
- [ ] 实现 JSON/Text/FormData/x-www-form-urlencoded 编辑界面
- [ ] 集成 Monaco Editor 提供脚本编辑与 JSON 高亮

### 阶段五：高级功能（第 5 天）

- [ ] 实现前后置脚本与 `pm` API
- [ ] 实现 Fetch API 导入（含剪贴板监听）
- [ ] 实现环境管理与切换
- [ ] 实现全局变量管理

### 阶段六： polish 与验证（第 6 天）

- [ ] 未保存提示、关闭确认
- [ ] 响应面板优化（状态、耗时、Body/Headers/Cookies）
- [ ] 内存/启动时间基准测试
- [ ] 构建 Windows 安装包

---

## 8. 架构决策记录（ADR）

### ADR-001：持久化层采用 JSON 文件而非 SQLite

- **背景**：任务书同时提到 JSON 文件和数据库。
- **决策**：以 JSON 文件为唯一持久化层，工作区即文件夹。
- **理由**：
  - 天然 Git 兼容，支持团队协作与版本回滚。
  - 减少运行时依赖，降低内存占用。
  - 符合“轻量级”定位。
- **代价**：大量小文件时目录遍历略慢，通过懒加载和缓存缓解。

### ADR-002：脚本引擎采用 goja（纯 Go）

- **背景**：需要在前/后置脚本中执行用户自定义 JS。
- **决策**：使用 `github.com/dop251/goja` 作为脚本引擎。
- **理由**：
  - 纯 Go，无需 CGO，跨平台构建简单。
  - 体积小，启动快。
  - 已内置 ES5+，足以覆盖需求。
- **代价**：不支持 Node.js 内置模块，需自行封装 `pm` API。

### ADR-003：前后端状态以前端 Pinia 为主

- **背景**：需要跟踪编辑态、dirty、当前环境等状态。
- **决策**：运行时状态由前端 Pinia 持有，Go 后端仅负责文件 IO 与请求执行。
- **理由**：
  - 减少前后端频繁通信。
  - 贴近 Vue 响应式模型，UI 更新自然。
  - 关闭前 dirty 检查由前端主导。
- **代价**：需要保证前后端文件状态一致性，保存成功后更新 store。

### ADR-004：剪贴板监听由后端负责

- **背景**：Fetch API 导入需要监控剪贴板。
- **决策**：Go 后端启动后台 goroutine 轮询剪贴板，检测到 fetch 代码通过 Wails Events 通知前端。
- **理由**：
  - 前端页面失焦后仍可监听。
  - 减少前端权限依赖。
- **代价**：1 秒轮询带来极低 CPU 开销。

---

## 9. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| JSON 文件过多导致目录加载慢 | 中 | 懒加载、缓存目录结构、提供工作区压缩导出 |
| 用户脚本死循环或内存泄漏 | 高 | 脚本执行超时 5 秒、限制内存、禁止访问文件系统 |
| WebSocket 大量消息导致 UI 卡顿 | 中 | 消息缓冲区限制、虚拟滚动、二进制消息可转文件 |
| 大响应体占用内存 | 高 | 默认 10MB 限制、提供“保存到文件”、流式读取 |
| Windows 安装包签名 | 低 | 使用 Wails 默认构建，暂不签名 |

---

## 10. 技术依赖

### Go 依赖

- `github.com/wailsapp/wails/v2`
- `github.com/imroc/req/v3` 或标准库 `net/http`
- `github.com/gorilla/websocket`
- `github.com/dop251/goja`
- `github.com/atotto/clipboard`
- `github.com/google/uuid`

### 前端依赖

- `vue@3`
- `element-plus`
- `pinia`
- `vue-router@4`
- `@element-plus/icons-vue`
- `monaco-editor`
- `jszip`（导出 zip）

---

*文档版本：v1.0*  
*作者：ApiClient Dev*  
*日期：2026-07-14*
