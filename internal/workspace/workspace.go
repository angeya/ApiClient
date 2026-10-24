package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ApiClient/internal/models"

	"github.com/google/uuid"
)

const settingsDir = ".apiclient"
const settingsFile = "settings.json"
const dirMetaFile = "_dir.meta.json"

// DefaultRequestTimeoutMs 请求默认超时时间（毫秒）。
const DefaultRequestTimeoutMs = 30000

// DefaultResponseRenderLimitBytes 响应内容默认渲染上限（10MB）。
// 超过该上限时前端不渲染响应正文，只提示下载后浏览。
const DefaultResponseRenderLimitBytes = int64(10 * 1024 * 1024)

// Manager handles workspace file IO.
type Manager struct {
	root string
}

// NewManager creates a workspace manager.
func NewManager(root string) *Manager {
	return &Manager{root: root}
}

// Root returns the workspace root path.
func (m *Manager) Root() string {
	return m.root
}

// EnsureWorkspace creates required directories and files.
func (m *Manager) EnsureWorkspace() error {
	if err := os.MkdirAll(m.root, 0755); err != nil {
		return err
	}
	settingsPath := filepath.Join(m.root, settingsDir, settingsFile)
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		settings := models.Settings{
			Version:                  "1.0.0",
			Theme:                    "dark-1",
			RequestTimeoutMs:         DefaultRequestTimeoutMs,
			ResponseRenderLimitBytes: DefaultResponseRenderLimitBytes,
			GlobalHeaders:            []models.Header{},
			GlobalVariables:          []models.Variable{},
			AutoRenameOnSave:         true,
		}
		if err := m.SaveSettings(settings); err != nil {
			return err
		}
	}
	return nil
}

// LoadSettings loads global settings.
func (m *Manager) LoadSettings() (models.Settings, error) {
	path := filepath.Join(m.root, settingsDir, settingsFile)
	var s models.Settings
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSettings(), nil
		}
		return s, err
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	// 兼容旧版 settings.json：缺失或非法的配置项回落到默认值
	if s.RequestTimeoutMs <= 0 {
		s.RequestTimeoutMs = DefaultRequestTimeoutMs
	}
	if s.ResponseRenderLimitBytes <= 0 {
		s.ResponseRenderLimitBytes = DefaultResponseRenderLimitBytes
	}
	// 兼容旧版 settings.json：字段缺失时默认开启保存时自动命名
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["autoRenameOnSave"]; !ok {
			s.AutoRenameOnSave = true
		}
	}
	return s, nil
}

// defaultSettings 返回一份包含全部默认值的设置，避免各处重复字面量。
func defaultSettings() models.Settings {
	return models.Settings{
		Version:                  "1.0.0",
		Theme:                    "dark-1",
		RequestTimeoutMs:         DefaultRequestTimeoutMs,
		ResponseRenderLimitBytes: DefaultResponseRenderLimitBytes,
		GlobalHeaders:            []models.Header{},
		GlobalVariables:          []models.Variable{},
		AutoRenameOnSave:         true,
	}
}

// SaveSettings saves global settings.
func (m *Manager) SaveSettings(s models.Settings) error {
	path := filepath.Join(m.root, settingsDir, settingsFile)
	return writeJSON(path, s)
}

// LoadTree loads the directory/request tree.
func (m *Manager) LoadTree() ([]models.ApiNode, error) {
	return m.loadTreeDir(m.root, "")
}

func (m *Manager) loadTreeDir(dir string, rel string) ([]models.ApiNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	// 预过滤有效条目，保持索引顺序
	type validEntry struct {
		name      string
		childRel  string
		childPath string
		isDir     bool
	}
	var valid []validEntry
	for _, entry := range entries {
		name := entry.Name()
		if name == settingsDir || strings.HasPrefix(name, ".") || name == dirMetaFile {
			continue
		}
		if !entry.IsDir() && filepath.Ext(name) != ".json" {
			continue
		}
		valid = append(valid, validEntry{
			name:      name,
			childRel:  filepath.Join(rel, name),
			childPath: filepath.Join(dir, name),
			isDir:     entry.IsDir(),
		})
	}
	if len(valid) == 0 {
		return nil, nil
	}

	// 并行加载各条目数据，按索引收集以保持排序顺序
	results := make([]models.ApiNode, len(valid))
	var wg sync.WaitGroup
	for i, ve := range valid {
		wg.Add(1)
		go func(idx int, e validEntry) {
			defer wg.Done()
			if e.isDir {
				children, _ := m.loadTreeDir(e.childPath, e.childRel)
				meta, _ := m.LoadDirMeta(e.childRel)
				results[idx] = models.ApiNode{
					ID:       meta.ID,
					Type:     "dir",
					Name:     e.name,
					Path:     e.childRel,
					Children: children,
				}
			} else {
				req, err := m.LoadRequest(e.childRel)
				if err != nil {
					return
				}
				results[idx] = models.ApiNode{
					ID:          req.ID,
					Type:        "request",
					Name:        req.Name,
					Method:      req.Method,
					RequestType: req.Type,
					Path:        e.childRel,
					URL:         req.URL,
					CreatedAt:   req.CreatedAt,
				}
			}
		}(i, ve)
	}
	wg.Wait()

	// 过滤掉未填充的零值节点（加载失败的条目）
	nodes := make([]models.ApiNode, 0, len(valid))
	for _, n := range results {
		if n.Type != "" {
			nodes = append(nodes, n)
		}
	}

	// 排序：目录在前；请求按创建时间升序，新增接口默认排在最后；
	// 创建时间相同（含旧数据无时间戳）时保持原有文件名顺序
	sort.SliceStable(nodes, func(i, j int) bool {
		aDir, bDir := nodes[i].Type == "dir", nodes[j].Type == "dir"
		if aDir != bDir {
			return aDir
		}
		if !aDir && nodes[i].CreatedAt != nodes[j].CreatedAt {
			return nodes[i].CreatedAt < nodes[j].CreatedAt
		}
		return false
	})
	return nodes, nil
}

// LoadDirMeta loads directory metadata.
func (m *Manager) LoadDirMeta(rel string) (models.DirMeta, error) {
	path := filepath.Join(m.root, rel, dirMetaFile)
	var meta models.DirMeta
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return models.DirMeta{ID: uuid.New().String(), Name: filepath.Base(rel)}, nil
		}
		return meta, err
	}
	err = json.Unmarshal(data, &meta)
	if meta.ID == "" {
		meta.ID = uuid.New().String()
	}
	return meta, err
}

// SaveDirMeta saves directory metadata.
func (m *Manager) SaveDirMeta(rel string, meta models.DirMeta) error {
	path := filepath.Join(m.root, rel, dirMetaFile)
	return writeJSON(path, meta)
}

// LoadRequest loads a request file.
func (m *Manager) LoadRequest(rel string) (models.Request, error) {
	path := filepath.Join(m.root, rel)
	var req models.Request
	data, err := os.ReadFile(path)
	if err != nil {
		return req, err
	}
	err = json.Unmarshal(data, &req)
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	return req, err
}

// SaveRequest saves a request file.
func (m *Manager) SaveRequest(rel string, req models.Request) error {
	path := filepath.Join(m.root, rel)
	// 前端保存不携带创建时间，保留文件中已有的值，避免排序信息丢失
	if req.CreatedAt == 0 {
		if old, err := m.LoadRequest(rel); err == nil {
			req.CreatedAt = old.CreatedAt
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return writeJSON(path, req)
}

// DeleteNode deletes a request or directory.
func (m *Manager) DeleteNode(rel string, isDir bool) error {
	path := filepath.Join(m.root, rel)
	if isDir {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// CreateDir creates a new directory with metadata.
func (m *Manager) CreateDir(parentRel string, name string) (string, error) {
	safeName := safeFileName(name)
	rel := filepath.Join(parentRel, safeName)
	path := filepath.Join(m.root, rel)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	meta := models.DirMeta{
		ID:   uuid.New().String(),
		Name: name,
	}
	if err := m.SaveDirMeta(rel, meta); err != nil {
		return "", err
	}
	return rel, nil
}

// CreateRequest creates a new request file.
func (m *Manager) CreateRequest(parentRel string, name string, reqType string) (string, error) {
	id := uuid.New().String()
	fileName := id + ".json"
	rel := filepath.Join(parentRel, fileName)
	path := filepath.Join(m.root, rel)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("request already exists: %s", rel)
	}
	method := "GET"
	if reqType == "websocket" {
		method = "WS"
	}
	req := models.Request{
		ID:        id,
		Type:      reqType,
		Name:      name,
		Method:    method,
		URL:       "",
		BodyType:  "none",
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := m.SaveRequest(rel, req); err != nil {
		return "", err
	}
	return rel, nil
}

// CloneRequest creates a copy of an existing request with a new ID.
func (m *Manager) CloneRequest(rel string, newName string) (string, error) {
	req, err := m.LoadRequest(rel)
	if err != nil {
		return "", err
	}
	parentRel := filepath.Dir(rel)
	id := uuid.New().String()
	fileName := id + ".json"
	newRel := filepath.Join(parentRel, fileName)
	path := filepath.Join(m.root, newRel)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("request already exists: %s", newRel)
	}
	req.ID = id
	req.Name = newName
	req.CreatedAt = time.Now().UnixMilli()
	if err := m.SaveRequest(newRel, req); err != nil {
		return "", err
	}
	return newRel, nil
}

// Rename renames a node.
func (m *Manager) Rename(rel string, newName string, isDir bool) (string, error) {
	if isDir {
		oldPath := filepath.Join(m.root, rel)
		newSafe := safeFileName(newName)
		newRel := filepath.Join(filepath.Dir(rel), newSafe)
		newPath := filepath.Join(m.root, newRel)
		if err := os.Rename(oldPath, newPath); err != nil {
			return "", err
		}
		meta, _ := m.LoadDirMeta(newRel)
		meta.Name = newName
		_ = m.SaveDirMeta(newRel, meta)
		return newRel, nil
	}
	req, err := m.LoadRequest(rel)
	if err != nil {
		return "", err
	}
	req.Name = newName
	if err := m.SaveRequest(rel, req); err != nil {
		return "", err
	}
	return rel, nil
}

// MoveNode moves a request or directory from one parent to another.
func (m *Manager) MoveNode(srcRel string, destDirRel string) error {
	srcPath := filepath.Join(m.root, srcRel)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source not found: %s", srcRel)
	}
	name := filepath.Base(srcRel)
	destRel := filepath.Join(destDirRel, name)
	destPath := filepath.Join(m.root, destRel)
	if srcRel == destRel {
		return nil
	}
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("target already exists: %s", destRel)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.Rename(srcPath, destPath)
}

// ReplacePrefixRel replaces the old prefix with the new prefix in a relative path.
func ReplacePrefixRel(rel, oldPrefix, newPrefix string) string {
	if oldPrefix == "" {
		return filepath.Join(newPrefix, rel)
	}
	if !strings.HasPrefix(rel, oldPrefix) {
		return rel
	}
	return filepath.Join(newPrefix, strings.TrimPrefix(rel, oldPrefix))
}

// ImportRequest imports a request JSON into the workspace.
func (m *Manager) ImportRequest(parentRel string, data []byte) (string, error) {
	var req models.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return "", err
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Name == "" {
		req.Name = "Imported Request"
	}
	if req.CreatedAt == 0 {
		req.CreatedAt = time.Now().UnixMilli()
	}
	rel := filepath.Join(parentRel, safeFileName(req.Name)+".json")
	path := filepath.Join(m.root, rel)
	if _, err := os.Stat(path); err == nil {
		rel = filepath.Join(parentRel, safeFileName(req.Name)+"-"+uuid.New().String()[:8]+".json")
	}
	return rel, m.SaveRequest(rel, req)
}

// ExportRequest exports a request to JSON bytes.
func (m *Manager) ExportRequest(rel string) ([]byte, error) {
	req, err := m.LoadRequest(rel)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(req, "", "  ")
}

// writeJSON writes a JSON file with indentation.
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func safeFileName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
