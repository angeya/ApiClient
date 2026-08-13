package variables

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ApiClient/internal/models"
)

var varRegex = regexp.MustCompile(`\{\{\s*([^}|\s]+)(?:\s*\|\s*([^}]+))?\s*\}\}`)

// Scope holds variable sources ordered by priority (low to high).
type Scope struct {
	Global    map[string]string
	Directory map[string]string
	Request   map[string]string
}

// NewScope creates a scope from settings and request metadata.
func NewScope(settings models.Settings, dir models.DirMeta, req models.Request) Scope {
	return Scope{
		Global:    toMap(settings.GlobalVariables),
		Directory: toMap(dir.Variables),
		Request:   toMap(extractRequestVars(req)),
	}
}

// Resolve replaces {{var}} placeholders in input using scope.
func Resolve(input string, scope Scope) string {
	return varRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := varRegex.FindStringSubmatch(match)
		key := parts[1]
		defaultVal := parts[2]
		val, ok := lookup(key, scope)
		if !ok {
			if defaultVal != "" {
				return strings.TrimSpace(defaultVal)
			}
			return match
		}
		return val
	})
}

// ResolveHeaders resolves header values.
func ResolveHeaders(headers []models.Header, scope Scope) []models.Header {
	var out []models.Header
	for _, h := range headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		out = append(out, models.Header{
			Key:         Resolve(h.Key, scope),
			Value:       Resolve(h.Value, scope),
			Enabled:     h.Enabled,
			Description: h.Description,
		})
	}
	return out
}

// ResolveQuery resolves query parameters.
func ResolveQuery(params []models.QueryParam, scope Scope) []models.QueryParam {
	var out []models.QueryParam
	for _, p := range params {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		out = append(out, models.QueryParam{
			Key:         Resolve(p.Key, scope),
			Value:       Resolve(p.Value, scope),
			Enabled:     p.Enabled,
			Description: p.Description,
		})
	}
	return out
}

// ResolveFormData resolves form field keys and text values.
func ResolveFormData(fields []models.FormField, scope Scope) []models.FormField {
	var out []models.FormField
	for _, f := range fields {
		if !f.Enabled || strings.TrimSpace(f.Key) == "" {
			continue
		}
		resolved := models.FormField{
			Key:         Resolve(f.Key, scope),
			Value:       f.Value,
			Type:        f.Type,
			Enabled:     f.Enabled,
			Description: f.Description,
		}
		if f.Type == "text" {
			resolved.Value = Resolve(f.Value, scope)
		}
		out = append(out, resolved)
	}
	return out
}

func lookup(key string, scope Scope) (string, bool) {
	if strings.HasPrefix(key, "$") {
		return builtIn(key), true
	}
	if v, ok := scope.Request[key]; ok {
		return v, true
	}
	if v, ok := scope.Directory[key]; ok {
		return v, true
	}
	if v, ok := scope.Global[key]; ok {
		return v, true
	}
	return "", false
}

func builtIn(key string) string {
	switch key {
	case "$timestamp":
		return strconv.FormatInt(time.Now().Unix(), 10)
	case "$timestampMs":
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	case "$isoDate":
		return time.Now().Format(time.RFC3339)
	case "$randomInt":
		return strconv.Itoa(int(time.Now().UnixNano() % 100000))
	default:
		return fmt.Sprintf("{%s}", key)
	}
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

func extractRequestVars(req models.Request) []models.Variable {
	return nil
}
