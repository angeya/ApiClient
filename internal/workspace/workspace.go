package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ApiClient/internal/models"

	"github.com/google/uuid"
)

const settingsDir = ".apiclient"
const settingsFile = "settings.json"
const dirMetaFile = "_dir.meta.json"

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
			Version:          "1.0.0",
			Theme:            "dark-1",
			RequestTimeoutMs: 30000,
			GlobalHeaders:    []models.Header{},
			GlobalVariables:  []models.Variable{},
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
			return models.Settings{Version: "1.0.0", Theme: "dark-1", RequestTimeoutMs: 30000}, nil
		}
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
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
	var nodes []models.ApiNode
	for _, entry := range entries {
		name := entry.Name()
		if name == settingsDir || strings.HasPrefix(name, ".") || name == dirMetaFile {
			continue
		}
		childRel := filepath.Join(rel, name)
		childPath := filepath.Join(dir, name)
		if entry.IsDir() {
			children, err := m.loadTreeDir(childPath, childRel)
			if err != nil {
				return nil, err
			}
			meta, _ := m.LoadDirMeta(childRel)
			nodes = append(nodes, models.ApiNode{
				ID:       meta.ID,
				Type:     "dir",
				Name:     name,
				Path:     childRel,
				Children: children,
			})
		} else if filepath.Ext(name) == ".json" {
			req, err := m.LoadRequest(childRel)
			if err != nil {
				continue
			}
			nodes = append(nodes, models.ApiNode{
				ID:          req.ID,
				Type:        "request",
				Name:        req.Name,
				Method:      req.Method,
				RequestType: req.Type,
				Path:        childRel,
				URL:         req.URL,
			})
		}
	}
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
		ID:       id,
		Type:     reqType,
		Name:     name,
		Method:   method,
		URL:      "",
		BodyType: "none",
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
