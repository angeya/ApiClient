export interface Variable {
  key: string
  value: string
  enabled: boolean
  description: string
}

export interface Header {
  key: string
  value: string
  enabled: boolean
  description: string
}

export interface QueryParam {
  key: string
  value: string
  enabled: boolean
  description: string
}

export interface FormField {
  key: string
  value: string
  type: 'text' | 'file'
  enabled: boolean
  description: string
}

export interface Request {
  id: string
  type: 'http' | 'websocket'
  name: string
  method: string
  url: string
  headers: Header[]
  query: QueryParam[]
  bodyType: 'none' | 'json' | 'text' | 'form-data' | 'x-www-form-urlencoded'
  body: string
  formData: FormField[]
  preRequestScript: string
  postResponseScript: string
  description: string
  /** 被禁用的继承请求头 key 列表，用于在单个请求中关闭全局/目录继承头 */
  disabledInheritedHeaderKeys?: string[]
}

export interface DirMeta {
  id: string
  name: string
  headers: Header[]
  variables: Variable[]
}

export interface Settings {
  version: string
  theme: string
  requestTimeoutMs: number
  globalHeaders: Header[]
  globalVariables: Variable[]
}

export interface WorkspaceMigrationResult {
  sourcePath: string
  targetPath: string
  copiedFiles: number
  skippedFiles: number
  createdDirs: number
  failedFiles: number
  failureSamples: string[]
  message: string
}

export interface HttpResponse {
  statusCode: number
  status: string
  headers: Record<string, string[]>
  cookies: string[]
  body: string
  timeMs: number
  serverTimeMs?: number
  transferTimeMs?: number
  sizeBytes: number
  contentType: string
}

export interface ApiNode {
  id: string
  type: 'dir' | 'request'
  name: string
  method?: string
  requestType?: string
  path: string
  url?: string
  children?: ApiNode[]
}

export interface FetchParseResult {
  url: string
  method: string
  headers: Record<string, string>
  body: string
}

export interface RequestTab {
  id: string
  rel: string
  name: string
  dirty: boolean
}
