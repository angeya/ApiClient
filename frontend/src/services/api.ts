import {
  OpenWorkspace,
  GetDefaultWorkspaceRoot,
  GetWorkspaceRoot,
  LoadTree,
  LoadSettings,
  SaveSettings,
  LoadRequest,
  SaveRequest,
  CreateDir,
  CreateRequest,
  CloneRequest,
  DeleteNode,
  Rename,
  LoadDirMeta,
  SaveDirMeta,
  ResolveVariables,
  PreviewRequest,
  SendHttpRequest,
  CancelRequest,
  ConnectWebSocket,
  SendWebSocketMessage,
  DisconnectWebSocket,
  ParseFetch,
  ImportRequest,
  ExportRequest,
  MoveNode,
  AllowClose,
  OpenInExplorer,
  PickWorkspaceDir,
  ChangeWorkspace,
  StartLoadTest,
  StopLoadTest,
  SaveResponseBody,
  PickFile,
} from '../../wailsjs/go/main/App'
import type { models } from '../../wailsjs/go/models'

export type HttpResponse = models.HttpResponse

export const api = {
  openWorkspace: OpenWorkspace,
  getDefaultWorkspaceRoot: GetDefaultWorkspaceRoot,
  getWorkspaceRoot: GetWorkspaceRoot,
  loadTree: LoadTree,
  loadSettings: LoadSettings,
  saveSettings: SaveSettings,
  loadRequest: LoadRequest,
  saveRequest: SaveRequest,
  createDir: CreateDir,
  createRequest: CreateRequest,
  cloneRequest: CloneRequest,
  deleteNode: DeleteNode,
  rename: Rename,
  loadDirMeta: LoadDirMeta,
  saveDirMeta: SaveDirMeta,
  resolveVariables: ResolveVariables,
  previewRequest: PreviewRequest,
  sendHttpRequest: SendHttpRequest,
  cancelRequest: CancelRequest,
  connectWebSocket: ConnectWebSocket,
  sendWebSocketMessage: SendWebSocketMessage,
  disconnectWebSocket: DisconnectWebSocket,
  parseFetch: ParseFetch,
  importRequest: ImportRequest,
  exportRequest: ExportRequest,
  moveNode: MoveNode,
  allowClose: AllowClose,
  openInExplorer: OpenInExplorer,
  pickWorkspaceDir: PickWorkspaceDir,
  changeWorkspace: ChangeWorkspace,
  startLoadTest: StartLoadTest,
  stopLoadTest: StopLoadTest,
  saveResponseBody: SaveResponseBody,
  pickFile: PickFile,
}

export function responseBodyToString(
  body: number[] | Uint8Array | ArrayBuffer | string | null | undefined
): string {
  if (body === null || body === undefined) {
    return ''
  }
  if (typeof body === 'string') {
    return body
  }
  if (body instanceof ArrayBuffer) {
    return new TextDecoder().decode(new Uint8Array(body))
  }
  if (ArrayBuffer.isView(body)) {
    return new TextDecoder().decode(body as Uint8Array)
  }
  if (Array.isArray(body)) {
    return new TextDecoder().decode(new Uint8Array(body))
  }
  return String(body)
}

export function responseBodyToJson(
  body: number[] | Uint8Array | ArrayBuffer | string | null | undefined
): unknown {
  try {
    return JSON.parse(responseBodyToString(body))
  } catch {
    return null
  }
}
