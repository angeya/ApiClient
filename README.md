# ApiClient

A lightweight, local-first desktop API client for testing and debugging **HTTP** and **WebSocket** interfaces. No login required — all request data is stored in a local workspace directory. Built with [Wails v2](https://wails.io) (Go + Vue 3).

> 面向开发者的轻量级 API 调试客户端，专注于 HTTP 与 WebSocket 接口的快速测试与调试。本地优先，无需登录，所有数据保存在本地工作区。

## Features | 功能

- **HTTP requests** — `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`
- **WebSocket debugging** — `WS`/`WSS` connections with send/receive
- **Multiple body types** — `none`, `json`, `text`, `form-data`, `x-www-form-urlencoded`
- **Query parameters & headers** — table-based editors with per-row enable toggles
- **Inherited headers** — global headers + directory-level headers, individually toggleable per request
- **Variables** — global, directory-level and request-level variables with `{{varName}}` resolution in URL, headers, query, and body
- **Pre/Post scripts** — JavaScript (Goja) scripts to modify requests, extract response data, and persist global variables
- **Load testing** — configurable concurrency and rounds/duration, with live progress (RPS, throughput, latency stats) and optional logging
- **Workspace management** — tree-structured request organization, drag-and-drop sorting, rename, clone, delete
- **Workspace migration** — switch the workspace directory and migrate data with a summary report
- **Import**
  - `fetch` code paste (auto-detected from clipboard)
  - OpenAPI / Swagger JSON
- **Request preview** — view the fully resolved request (URL, query, headers, body) before sending
- **Response panel** — status, headers, cookies, body, timing, size, content-type, and script logs
- **Themes** — Dark 1, Dark 2, Warm
- **Keyboard shortcut** — `Ctrl/Cmd + S` to save the current request

## Tech Stack | 技术栈

| Layer | Technology |
|-------|-----------|
| Desktop framework | Wails v2 |
| Frontend | Vue 3 + TypeScript + Vite |
| UI components | Element Plus |
| Backend | Go |
| Script engine | Goja (JavaScript) |
| State management | Pinia |

## Project Structure | 项目结构

```
ApiClient/
├── app.go                      # Wails app entry & core bindings
├── main.go                     # Program entry
├── download.go                 # Download helpers
├── wails.json                  # Wails project config
├── build/                      # Build assets (icons, manifests, logo script)
├── docs/                       # Design documents
├── frontend/                   # Vue 3 frontend
│   ├── src/
│   │   ├── components/         # Vue components
│   │   ├── composables/        # Vue composables
│   │   ├── router/             # Vue Router
│   │   ├── services/           # Wails API service wrapper
│   │   ├── stores/             # Pinia stores
│   │   ├── styles/             # Global styles & themes
│   │   ├── types/              # TypeScript type definitions
│   │   ├── utils/              # Utilities (helpers, swagger-import)
│   │   └── views/              # Page views
│   └── wailsjs/                # Auto-generated Wails bindings
└── internal/                   # Go internal packages
    ├── clipboard/              # Clipboard monitoring (fetch detection)
    ├── fetch/                  # fetch() code parser
    ├── filedialog/             # Native file dialogs
    ├── hooks/                  # Pre/Post script engine (Goja)
    ├── loadtest/               # Load test engine
    ├── models/                 # Data models
    ├── request/                # HTTP & WebSocket executors
    ├── utils/                  # Header utilities
    ├── variables/              # Variable resolution
    └── workspace/              # Workspace file IO
```

## Getting Started | 快速开始

### Prerequisites | 环境要求

- Go 1.23+
- Node.js 18+
- Wails CLI v2.12+ (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Install dependencies | 安装依赖

```bash
# Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Frontend dependencies
cd frontend && npm install
```

### Development | 开发运行

```bash
wails dev
```

A Vite dev server provides hot reload of frontend changes. Backend Go methods are callable from the frontend during development.

### Production build | 构建生产版本

```bash
wails build
```

The redistributable executable is output to `build/bin/`.

## Workspace Layout | 工作区结构

A workspace is a regular directory containing:

- `*.json` — request files (one per request)
- `_dir.meta.json` — directory-level metadata (inherited headers & variables)
- `.apiclient/settings.json` — global settings (theme, timeout, global headers/variables)
- `.log/` — load test logs (when enabled)

## Script API | 脚本 API

Scripts run on the Goja JavaScript engine with a `pm` object:

```javascript
// Pre-request: set a global variable
pm.setGlobal('token', 'abc123')
pm.log('token set')

// Post-response: extract from the response body
const json = pm.response.json()
pm.setGlobal('userId', json.id)
pm.log('status:', pm.response.statusCode)
```

| Method | Description |
|--------|-------------|
| `pm.log(...args)` | Append a log line (shown in the response panel) |
| `pm.setGlobal(key, value)` | Set a global variable (persisted to settings) |
| `pm.getGlobal(key)` | Get a global variable |
| `pm.setEnv(key, value)` | Set an environment variable |
| `pm.getEnv(key)` | Get an environment variable |
| `pm.request` | The request object (`url`, `method`, `headers`, `body`) — pre-request only |
| `pm.response` | The response object (`statusCode`, `status`, `headers`, `body`, `timeMs`, `json()`) — post-response only |
| `pm.responseJSON()` | Parse the response body as JSON |

## License | 许可证

Internal tool. Unauthorized commercial distribution is prohibited.
