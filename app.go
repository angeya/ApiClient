package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ApiClient/internal/clipboard"
	"ApiClient/internal/fetch"
	"ApiClient/internal/hooks"
	"ApiClient/internal/loadtest"
	"ApiClient/internal/models"
	"ApiClient/internal/request"
	"ApiClient/internal/utils"
	"ApiClient/internal/variables"
	"ApiClient/internal/workspace"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultWorkspaceName = "ApiClientWorkspace"
	appConfigDirName     = ".apiclient"
	appConfigFileName    = "app.json"
	loadTestLogDirName   = ".log"
)

type appConfig struct {
	WorkspaceRoot string `json:"workspaceRoot"`
}

// App provides Wails bindings for the API client.
type App struct {
	ctx            context.Context
	workspaceRoot  string
	workspace      *workspace.Manager
	httpExecutor   *request.HttpExecutor
	loadTestEngine *loadtest.Engine
	wsManager      *request.WSManager
	hookEngine     *hooks.ScriptEngine
	clipMonitor    *clipboard.Monitor
	allowClose     bool
	reqCancels     map[string]context.CancelFunc
	reqCancelsMu   sync.Mutex
}

// NewApp creates a new App application struct.
func NewApp() *App {
	a := &App{
		reqCancels: make(map[string]context.CancelFunc),
	}
	a.httpExecutor = request.NewHttpExecutor()
	a.hookEngine = hooks.NewScriptEngine()
	a.wsManager = request.NewWSManager(func() context.Context { return a.ctx })
	a.clipMonitor = clipboard.NewMonitor()
	return a
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.clipMonitor.SetContext(ctx)
	a.clipMonitor.Start()
	a.loadTestEngine = loadtest.NewEngine(a.httpExecutor, func(event string, data ...any) {
		wailsRuntime.EventsEmit(ctx, event, data...)
	})
}

// shutdown is called when the app is shutting down.
func (a *App) shutdown(ctx context.Context) {
	a.clipMonitor.Stop()
}

// beforeClose is called before the window closes.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.allowClose {
		return false
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "app:close-request")
	}
	return true
}

// AllowClose allows the application window to close.
func (a *App) AllowClose() {
	a.allowClose = true
}

// GetDefaultWorkspaceRoot returns the persisted workspace root, or the app-local default.
func (a *App) GetDefaultWorkspaceRoot() string {
	cfg, err := a.loadAppConfig()
	if err == nil && strings.TrimSpace(cfg.WorkspaceRoot) != "" {
		return cfg.WorkspaceRoot
	}
	return filepath.Join(a.appBaseDir(), defaultWorkspaceName)
}

// OpenWorkspace opens a workspace directory.
func (a *App) OpenWorkspace(root string) error {
	if root == "" {
		return fmt.Errorf("workspace root cannot be empty")
	}
	a.workspaceRoot = root
	a.workspace = workspace.NewManager(root)
	if err := a.workspace.EnsureWorkspace(); err != nil {
		return err
	}
	cfg, _ := a.loadAppConfig()
	cfg.WorkspaceRoot = root
	if err := a.saveAppConfig(cfg); err != nil {
		return err
	}
	return nil
}

// GetWorkspaceRoot returns the current workspace root.
func (a *App) GetWorkspaceRoot() string {
	return a.workspaceRoot
}

// PickWorkspaceDir opens a native directory picker for workspace selection.
func (a *App) PickWorkspaceDir() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未启动")
	}
	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择工作区目录",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// ChangeWorkspace migrates current workspace data into a new directory and opens it.
func (a *App) ChangeWorkspace(targetRoot string) (models.WorkspaceMigrationResult, error) {
	targetRoot = strings.TrimSpace(targetRoot)
	if targetRoot == "" {
		return models.WorkspaceMigrationResult{}, fmt.Errorf("workspace root cannot be empty")
	}
	if a.workspace == nil || strings.TrimSpace(a.workspaceRoot) == "" {
		if err := a.OpenWorkspace(targetRoot); err != nil {
			return models.WorkspaceMigrationResult{}, err
		}
		return models.WorkspaceMigrationResult{
			TargetPath: targetRoot,
			Message:    "已打开新的工作区",
		}, nil
	}

	sourceRoot, err := filepath.Abs(a.workspaceRoot)
	if err != nil {
		sourceRoot = a.workspaceRoot
	}
	targetAbs, err := filepath.Abs(targetRoot)
	if err != nil {
		return models.WorkspaceMigrationResult{}, err
	}

	result := models.WorkspaceMigrationResult{
		SourcePath: sourceRoot,
		TargetPath: targetAbs,
	}
	if samePath(sourceRoot, targetAbs) {
		result.Message = "工作区目录未变化，无需迁移"
		return result, nil
	}

	if err := migrateWorkspace(sourceRoot, targetAbs, &result); err != nil {
		return result, err
	}
	if err := a.OpenWorkspace(targetAbs); err != nil {
		return result, err
	}
	result.Message = fmt.Sprintf("迁移完成：复制 %d 个文件，跳过 %d 个同名文件，失败 %d 个文件", result.CopiedFiles, result.SkippedFiles, result.FailedFiles)
	return result, nil
}

// OpenInExplorer opens the given path in the system file explorer.
func (a *App) OpenInExplorer(path string) error {
	if path == "" {
		return fmt.Errorf("路径不能为空")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("目录不存在: %s", path)
		}
		return fmt.Errorf("无法访问目录: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", path)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("无法打开文件管理器: %w", err)
	}
	return nil
}

// PickFile opens a native file picker dialog and returns the selected file path.
// 用于在 form-data 文件字段中选择文件，避免浏览器 input[type=file] 拿不到完整路径的问题。
func (a *App) PickFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未启动")
	}
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择文件",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// LoadTree loads the workspace tree.
func (a *App) LoadTree() ([]models.ApiNode, error) {
	if a.workspace == nil {
		return nil, fmt.Errorf("workspace not opened")
	}
	return a.workspace.LoadTree()
}

// LoadSettings loads global settings.
func (a *App) LoadSettings() (models.Settings, error) {
	if a.workspace == nil {
		return models.Settings{}, fmt.Errorf("workspace not opened")
	}
	return a.workspace.LoadSettings()
}

// SaveSettings saves global settings.
func (a *App) SaveSettings(s models.Settings) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	return a.workspace.SaveSettings(s)
}

// LoadRequest loads a request by relative path.
func (a *App) LoadRequest(rel string) (models.Request, error) {
	if a.workspace == nil {
		return models.Request{}, fmt.Errorf("workspace not opened")
	}
	return a.workspace.LoadRequest(rel)
}

// SaveRequest saves a request.
func (a *App) SaveRequest(rel string, req models.Request) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	if req.BodyType == "json" && req.Body != "" {
		if err := request.ValidateJSONBody(req.Body); err != nil {
			return err
		}
	}
	return a.workspace.SaveRequest(rel, req)
}

// CreateDir creates a directory.
func (a *App) CreateDir(parentRel string, name string) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	return a.workspace.CreateDir(parentRel, name)
}

// CreateRequest creates a new request file.
func (a *App) CreateRequest(parentRel string, name string, reqType string) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	return a.workspace.CreateRequest(parentRel, name, reqType)
}

// CloneRequest creates a copy of an existing request.
func (a *App) CloneRequest(rel string, newName string) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	return a.workspace.CloneRequest(rel, newName)
}

// DeleteNode deletes a node.
func (a *App) DeleteNode(rel string, isDir bool) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	return a.workspace.DeleteNode(rel, isDir)
}

// Rename renames a node.
func (a *App) Rename(rel string, newName string, isDir bool) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	return a.workspace.Rename(rel, newName, isDir)
}

// MoveNode moves a node to a new parent directory.
func (a *App) MoveNode(srcRel string, destDirRel string) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	return a.workspace.MoveNode(srcRel, destDirRel)
}

// LoadDirMeta loads directory metadata.
func (a *App) LoadDirMeta(rel string) (models.DirMeta, error) {
	if a.workspace == nil {
		return models.DirMeta{}, fmt.Errorf("workspace not opened")
	}
	return a.workspace.LoadDirMeta(rel)
}

// ResolveVariables resolves variables in a string for preview.
func (a *App) ResolveVariables(input string, dirRel string, reqRel string) (string, error) {
	settings, _ := a.workspace.LoadSettings()
	dir, _ := a.workspace.LoadDirMeta(dirRel)
	req, _ := a.workspace.LoadRequest(reqRel)
	scope := variables.NewScope(settings, dir, req)
	return variables.Resolve(input, scope), nil
}

// SendHttpRequest sends an HTTP request.
func (a *App) SendHttpRequest(req models.Request, dirRel string) (models.HttpResponse, error) {
	var resp models.HttpResponse
	if a.workspace == nil {
		return resp, fmt.Errorf("workspace not opened")
	}

	reqCtx, cancel := context.WithCancel(a.ctx)
	a.reqCancelsMu.Lock()
	a.reqCancels[req.ID] = cancel
	a.reqCancelsMu.Unlock()
	defer func() {
		a.reqCancelsMu.Lock()
		delete(a.reqCancels, req.ID)
		a.reqCancelsMu.Unlock()
		cancel()
	}()

	settings, _ := a.workspace.LoadSettings()
	dir, _ := a.workspace.LoadDirMeta(dirRel)

	scope := variables.NewScope(settings, dir, req)
	resolvedURL := variables.Resolve(req.URL, scope)
	req.Query = variables.ResolveQuery(req.Query, scope)
	req.FormData = variables.ResolveFormData(req.FormData, scope)
	mergedHeaders := utils.MergeHeaders(settings.GlobalHeaders, dir.Headers, req.Headers, req.DisabledInheritedHeaderKeys)
	resolvedHeaders := variables.ResolveHeaders(mergedHeaders, scope)

	var body []byte
	var contentType string
	var err error
	if req.BodyType == "form-data" {
		body, contentType, err = request.BuildMultipartBody(req)
		if err != nil {
			return resp, err
		}
	} else {
		resolvedBody := variables.Resolve(req.Body, scope)
		body, contentType, err = request.BuildBody(models.Request{BodyType: req.BodyType, Body: resolvedBody, FormData: req.FormData})
		if err != nil {
			return resp, err
		}
	}

	// 无论是否有脚本，都需要把正确的 Content-Type 写入请求头，否则 form-data / urlencoded 等请求体无法被服务端正确解析。
	// form-data 与 urlencoded 必须强制覆盖（含 boundary），json/text 仅在用户未指定时补充默认值。
	if contentType != "" {
		forceCT := req.BodyType == "form-data" || req.BodyType == "x-www-form-urlencoded"
		resolvedHeaders = setContentType(resolvedHeaders, contentType, forceCT)
	}

	scriptReq := &hooks.ScriptRequest{
		URL:     resolvedURL,
		Method:  req.Method,
		Headers: headerMap(resolvedHeaders),
		Body:    string(body),
	}
	globalMap := toMap(settings.GlobalVariables)

	// 默认使用原始变量，前置脚本执行后会被覆盖
	newGlobal := globalMap
	// 仅在存在前置脚本时执行，避免空脚本也产生 PreScript 结果
	if req.PreRequestScript != "" {
		log.Printf("[pre-request] 开始执行前置脚本, 长度=%d, 请求ID=%s", len(req.PreRequestScript), req.ID)
		preStart := time.Now()
		var preLogs []string
		var modifiedReq *hooks.ScriptRequest
		var preErr error
		newGlobal, _, preLogs, modifiedReq, preErr = a.hookEngine.ExecutePre(reqCtx, req.PreRequestScript, scriptReq, globalMap, nil)
		preDuration := time.Since(preStart).Milliseconds()
		resp.PreScript = &models.ScriptResult{
			Success:    preErr == nil,
			DurationMs: preDuration,
			Logs:       preLogs,
		}
		if preErr != nil {
			// 脚本执行失败时记录错误，并保存已发生的变量变更；
			// 不返回 error，以便前端能拿到 resp.PreScript 进行展示
			log.Printf("[pre-request] 脚本执行失败: %v, 日志数=%d, 耗时=%dms", preErr, len(preLogs), preDuration)
			resp.PreScript.Error = preErr.Error()
			settings.GlobalVariables = fromMap(newGlobal)
			_ = a.workspace.SaveSettings(settings)
			return resp, nil
		}
		log.Printf("[pre-request] 脚本执行成功, 日志数=%d, 耗时=%dms", len(preLogs), preDuration)
		// 应用脚本对请求的修改
		resolvedURL = modifiedReq.URL
		req.Method = modifiedReq.Method
		body = []byte(modifiedReq.Body)
		resolvedHeaders = fromHeaderMap(modifiedReq.Headers, resolvedHeaders)
		if contentType != "" {
			resolvedHeaders = setContentType(resolvedHeaders, contentType, true)
		}
	}

	httpReq := models.HttpRequest{
		Request:         req,
		ResolvedURL:     resolvedURL,
		ResolvedHeaders: resolvedHeaders,
		ResolvedBody:    body,
		GlobalVariables: newGlobal,
	}

	// 根据全局设置应用请求超时，未设置或非法时使用默认 30 秒
	timeoutMs := settings.RequestTimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	execCtx, execCancel := context.WithTimeout(reqCtx, time.Duration(timeoutMs)*time.Millisecond)
	defer execCancel()

	preScript := resp.PreScript
	resp, err = a.httpExecutor.Execute(execCtx, httpReq)
	if err != nil {
		resp.PreScript = preScript
		return resp, err
	}
	resp.PreScript = preScript

	scriptResp := &hooks.ScriptResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Headers,
		Body:       string(resp.Body),
		TimeMs:     resp.TimeMs,
	}

	// 仅在存在后置脚本时执行，避免空脚本也产生 PostScript 结果
	if req.PostResponseScript != "" {
		log.Printf("[post-response] 开始执行后置脚本, 长度=%d, 请求ID=%s", len(req.PostResponseScript), req.ID)
		postStart := time.Now()
		finalGlobal, _, postLogs, postErr := a.hookEngine.ExecutePost(reqCtx, req.PostResponseScript, scriptResp, newGlobal, nil)
		postDuration := time.Since(postStart).Milliseconds()
		resp.PostScript = &models.ScriptResult{
			Success:    postErr == nil,
			DurationMs: postDuration,
			Logs:       postLogs,
		}
		if postErr != nil {
			log.Printf("[post-response] 脚本执行失败: %v, 日志数=%d, 耗时=%dms", postErr, len(postLogs), postDuration)
			resp.PostScript.Error = postErr.Error()
		} else {
			log.Printf("[post-response] 脚本执行成功, 日志数=%d, 耗时=%dms", len(postLogs), postDuration)
		}
		settings.GlobalVariables = fromMap(finalGlobal)
		_ = a.workspace.SaveSettings(settings)
	} else {
		// 没有后置脚本时，仍然需要保存前置脚本可能产生的变量变更
		settings.GlobalVariables = fromMap(newGlobal)
		_ = a.workspace.SaveSettings(settings)
	}

	return resp, nil
}

// CancelRequest cancels a pending HTTP request by its ID.
func (a *App) CancelRequest(id string) error {
	a.reqCancelsMu.Lock()
	cancel, ok := a.reqCancels[id]
	a.reqCancelsMu.Unlock()
	if !ok {
		return fmt.Errorf("请求不在进行中")
	}
	cancel()
	return nil
}

// StartLoadTest starts a load test for the given request.
func (a *App) StartLoadTest(req models.Request, dirRel string, config models.LoadTestConfig) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	if a.loadTestEngine == nil {
		return "", fmt.Errorf("load test engine not initialized")
	}

	settings, _ := a.workspace.LoadSettings()
	dir, _ := a.workspace.LoadDirMeta(dirRel)

	scope := variables.NewScope(settings, dir, req)
	resolvedURL := variables.Resolve(req.URL, scope)
	req.Query = variables.ResolveQuery(req.Query, scope)
	req.FormData = variables.ResolveFormData(req.FormData, scope)
	mergedHeaders := utils.MergeHeaders(settings.GlobalHeaders, dir.Headers, req.Headers, req.DisabledInheritedHeaderKeys)
	resolvedHeaders := variables.ResolveHeaders(mergedHeaders, scope)

	var body []byte
	var contentType string
	var err error
	if req.BodyType == "form-data" {
		body, contentType, err = request.BuildMultipartBody(req)
		if err != nil {
			return "", err
		}
	} else {
		resolvedBody := variables.Resolve(req.Body, scope)
		body, contentType, err = request.BuildBody(models.Request{BodyType: req.BodyType, Body: resolvedBody, FormData: req.FormData})
		if err != nil {
			return "", err
		}
	}
	if contentType != "" {
		forceCT := req.BodyType == "form-data" || req.BodyType == "x-www-form-urlencoded"
		resolvedHeaders = setContentType(resolvedHeaders, contentType, forceCT)
	}

	httpReq := models.HttpRequest{
		Request:         req,
		ResolvedURL:     resolvedURL,
		ResolvedHeaders: resolvedHeaders,
		ResolvedBody:    body,
		GlobalVariables: toMap(settings.GlobalVariables),
	}

	var logDir string
	if config.LogEnabled {
		logDir = filepath.Join(a.workspaceRoot, loadTestLogDirName, loadtest.SanitizeName(req.Name))
	}

	return a.loadTestEngine.Start(httpReq, config, logDir)
}

// StopLoadTest stops a running load test.
func (a *App) StopLoadTest(id string) error {
	if a.loadTestEngine == nil {
		return fmt.Errorf("load test engine not initialized")
	}
	return a.loadTestEngine.Stop(id)
}

// ConnectWebSocket connects to a WebSocket endpoint.
func (a *App) ConnectWebSocket(id string, req models.Request, dirRel string) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	settings, _ := a.workspace.LoadSettings()
	dir, _ := a.workspace.LoadDirMeta(dirRel)
	scope := variables.NewScope(settings, dir, req)
	url := variables.Resolve(req.URL, scope)
	mergedHeaders := utils.MergeHeaders(settings.GlobalHeaders, dir.Headers, req.Headers, req.DisabledInheritedHeaderKeys)
	resolvedHeaders := variables.ResolveHeaders(mergedHeaders, scope)
	headers := http.Header{}
	for _, h := range resolvedHeaders {
		headers.Set(h.Key, h.Value)
	}
	return a.wsManager.Connect(id, url, headers)
}

// SendWebSocketMessage sends a message over a WebSocket connection.
func (a *App) SendWebSocketMessage(id string, message string) error {
	return a.wsManager.Send(id, message)
}

// DisconnectWebSocket disconnects a WebSocket connection.
func (a *App) DisconnectWebSocket(id string) error {
	return a.wsManager.Disconnect(id)
}

// SaveDirMeta saves directory metadata.
func (a *App) SaveDirMeta(rel string, meta models.DirMeta) error {
	if a.workspace == nil {
		return fmt.Errorf("workspace not opened")
	}
	return a.workspace.SaveDirMeta(rel, meta)
}

// PreviewRequest returns the resolved request preview.
func (a *App) PreviewRequest(req models.Request, dirRel string) (models.RequestPreview, error) {
	var preview models.RequestPreview
	if a.workspace == nil {
		return preview, fmt.Errorf("workspace not opened")
	}
	settings, _ := a.workspace.LoadSettings()
	dir, _ := a.workspace.LoadDirMeta(dirRel)
	scope := variables.NewScope(settings, dir, req)
	mergedHeaders := utils.MergeHeaders(settings.GlobalHeaders, dir.Headers, req.Headers, req.DisabledInheritedHeaderKeys)

	req.Query = variables.ResolveQuery(req.Query, scope)
	req.FormData = variables.ResolveFormData(req.FormData, scope)
	resolvedBody := variables.Resolve(req.Body, scope)

	preview.URL = variables.Resolve(req.URL, scope)
	preview.Headers = variables.ResolveHeaders(mergedHeaders, scope)
	preview.Query = req.Query
	preview.BodyType = req.BodyType

	switch req.BodyType {
	case "form-data":
		body, ct, err := request.BuildMultipartBody(req)
		if err != nil {
			return preview, err
		}
		preview.Body = string(body)
		preview.ContentType = ct
		preview.FormData = req.FormData
	case "x-www-form-urlencoded":
		body, ct, err := request.BuildBody(models.Request{BodyType: req.BodyType, Body: resolvedBody, FormData: req.FormData})
		if err != nil {
			return preview, err
		}
		preview.Body = string(body)
		preview.ContentType = ct
	case "json", "text":
		preview.Body = resolvedBody
		preview.ContentType = request.ContentTypeFor(req.BodyType)
	default:
		preview.Body = resolvedBody
	}

	return preview, nil
}

// ParseFetch parses fetch API code.
func (a *App) ParseFetch(code string) (models.FetchParseResult, error) {
	return fetch.Parse(code)
}

// ImportRequest imports a request JSON.
func (a *App) ImportRequest(parentRel string, data []byte) (string, error) {
	if a.workspace == nil {
		return "", fmt.Errorf("workspace not opened")
	}
	return a.workspace.ImportRequest(parentRel, data)
}

// ExportRequest exports a request to JSON.
func (a *App) ExportRequest(rel string) ([]byte, error) {
	if a.workspace == nil {
		return nil, fmt.Errorf("workspace not opened")
	}
	return a.workspace.ExportRequest(rel)
}

func headerMap(headers []models.Header) map[string]string {
	m := make(map[string]string)
	for _, h := range headers {
		m[h.Key] = h.Value
	}
	return m
}

func fromHeaderMap(m map[string]string, base []models.Header) []models.Header {
	seen := make(map[string]bool)
	var out []models.Header
	for _, h := range base {
		if v, ok := m[h.Key]; ok {
			h.Value = v
			out = append(out, h)
			seen[h.Key] = true
		}
	}
	for k, v := range m {
		if !seen[k] {
			out = append(out, models.Header{Key: k, Value: v, Enabled: true})
		}
	}
	return out
}

func setContentType(headers []models.Header, ct string, force bool) []models.Header {
	for i := range headers {
		if strings.EqualFold(headers[i].Key, "Content-Type") {
			if force {
				headers[i].Value = ct
			}
			return headers
		}
	}
	return append(headers, models.Header{Key: "Content-Type", Value: ct, Enabled: true})
}

func toMap(vars []models.Variable) map[string]string {
	m := make(map[string]string)
	for _, v := range vars {
		if !v.Enabled || strings.TrimSpace(v.Key) == "" {
			continue
		}
		m[v.Key] = v.Value
	}
	return m
}

func fromMap(m map[string]string) []models.Variable {
	var vars []models.Variable
	for k, v := range m {
		vars = append(vars, models.Variable{Key: k, Value: v, Enabled: true})
	}
	return vars
}

func (a *App) appBaseDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func (a *App) appConfigPath() string {
	return filepath.Join(a.appBaseDir(), appConfigDirName, appConfigFileName)
}

func (a *App) loadAppConfig() (appConfig, error) {
	var cfg appConfig
	data, err := os.ReadFile(a.appConfigPath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func (a *App) saveAppConfig(cfg appConfig) error {
	path := a.appConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func samePath(a string, b string) bool {
	aa, err := filepath.Abs(a)
	if err == nil {
		a = aa
	}
	bb, err := filepath.Abs(b)
	if err == nil {
		b = bb
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func migrateWorkspace(sourceRoot string, targetRoot string, result *models.WorkspaceMigrationResult) error {
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return err
	}
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.FailedFiles++
			addFailureSample(result, fmt.Sprintf("%s: %v", path, walkErr))
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if samePath(path, targetRoot) {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil || rel == "." {
			return nil
		}
		targetPath := filepath.Join(targetRoot, rel)
		if d.IsDir() {
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				if err := os.MkdirAll(targetPath, 0755); err != nil {
					result.FailedFiles++
					addFailureSample(result, fmt.Sprintf("%s: %v", targetPath, err))
					return filepath.SkipDir
				}
				result.CreatedDirs++
			}
			return nil
		}
		if _, err := os.Stat(targetPath); err == nil {
			result.SkippedFiles++
			return nil
		} else if !os.IsNotExist(err) {
			result.FailedFiles++
			addFailureSample(result, fmt.Sprintf("%s: %v", targetPath, err))
			return nil
		}
		if err := copyFile(path, targetPath); err != nil {
			result.FailedFiles++
			addFailureSample(result, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		result.CopiedFiles++
		return nil
	})
}

func copyFile(sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, info.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func addFailureSample(result *models.WorkspaceMigrationResult, sample string) {
	if len(result.FailureSamples) < 5 {
		result.FailureSamples = append(result.FailureSamples, sample)
	}
}
