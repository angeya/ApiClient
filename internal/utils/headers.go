package utils

import (
	"strings"

	"ApiClient/internal/models"
)

// MergeHeaders merges headers from global, directory and request levels.
// Later levels override earlier ones for the same key (case-insensitive).
// disabledInheritedKeys can be used to explicitly remove inherited headers
// (from global or directory) for the current request.
func MergeHeaders(global, dir, req []models.Header, disabledInheritedKeys []string) []models.Header {
	disabled := make(map[string]bool, len(disabledInheritedKeys))
	for _, k := range disabledInheritedKeys {
		disabled[key(k)] = true
	}
	m := make(map[string]models.Header)
	for _, h := range global {
		if !h.Enabled || disabled[key(h.Key)] {
			continue
		}
		m[key(h.Key)] = h
	}
	for _, h := range dir {
		if !h.Enabled || disabled[key(h.Key)] {
			continue
		}
		m[key(h.Key)] = h
	}
	for _, h := range req {
		if !h.Enabled {
			continue
		}
		m[key(h.Key)] = h
	}
	out := make([]models.Header, 0, len(m))
	for _, h := range m {
		out = append(out, h)
	}
	return out
}

func key(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
