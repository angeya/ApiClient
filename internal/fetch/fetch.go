package fetch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ApiClient/internal/models"
)

var quoteClass = `"'` + "`" // matches ", ', `

// Parse extracts fetch call details from JavaScript code.
func Parse(code string) (models.FetchParseResult, error) {
	var result models.FetchParseResult
	code = strings.TrimSpace(code)
	if !strings.Contains(code, "fetch(") {
		return result, fmt.Errorf("no fetch call found")
	}

	// Extract URL
	urlRe := regexp.MustCompile(`fetch\(\s*[` + quoteClass + `]([^` + quoteClass + `]+)[` + quoteClass + `]`)
	if m := urlRe.FindStringSubmatch(code); len(m) > 1 {
		result.URL = m[1]
	}

	// Extract method
	methodRe := regexp.MustCompile(`["']?[Mm]ethod["']?\s*:\s*[` + quoteClass + `]([A-Za-z]+)[` + quoteClass + `]`)
	if m := methodRe.FindStringSubmatch(code); len(m) > 1 {
		result.Method = strings.ToUpper(m[1])
	} else {
		result.Method = "GET"
	}

	// Extract headers object
	headersRe := regexp.MustCompile(`["']?[Hh]eaders["']?\s*:\s*\{([^}]*)\}`)
	result.Headers = make(map[string]string)
	if m := headersRe.FindStringSubmatch(code); len(m) > 1 {
		headerBody := m[1]
		pairs := parseKeyValuePairs(headerBody)
		for k, v := range pairs {
			result.Headers[k] = v
		}
	}

	// Extract body
	bodyRe := regexp.MustCompile(`["']?[Bb]ody["']?\s*:\s*([\s\S]*?)(?:,\s*\n|\n\s*\})`)
	if m := bodyRe.FindStringSubmatch(code); len(m) > 1 {
		body := strings.TrimSpace(m[1])
		if body == "null" || body == "undefined" {
			result.Body = ""
		} else if len(body) > 1 && (body[0] == '"' || body[0] == '\'' || body[0] == '`') {
			result.Body = unquote(body)
		} else if body[0] == '{' || body[0] == '[' {
			result.Body = body
		} else if strings.HasPrefix(body, "JSON.stringify") {
			result.Body = extractJSONStringify(body)
		}
	}

	// Extract query parameters from URL
	// URL 本身保留完整查询字符串，Query 字段同时保存解析后的参数列表，
	// 由前端在导入时同步到 Query 表格，确保原始 URL 参数信息不丢失。
	result.Query = parseQueryParams(result.URL)

	return result, nil
}

func parseKeyValuePairs(s string) map[string]string {
	m := make(map[string]string)
	re := regexp.MustCompile(`[` + quoteClass + `]([^` + quoteClass + `]+)[` + quoteClass + `]\s*:\s*[` + quoteClass + `]([^` + quoteClass + `]+)[` + quoteClass + `]`)
	for _, match := range re.FindAllStringSubmatch(s, -1) {
		m[match[1]] = match[2]
	}
	return m
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	quote := s[0]
	if quote != '"' && quote != '\'' && quote != '`' {
		return s
	}
	if s[len(s)-1] != byte(quote) {
		return s
	}
	inner := s[1 : len(s)-1]
	inner = strings.ReplaceAll(inner, "\\n", "\n")
	inner = strings.ReplaceAll(inner, "\\\"", "\"")
	inner = strings.ReplaceAll(inner, "\\'", "'")
	return inner
}

func extractJSONStringify(s string) string {
	start := strings.Index(s, "(")
	end := strings.LastIndex(s, ")")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	arg := strings.TrimSpace(s[start+1 : end])
	if len(arg) > 1 && (arg[0] == '{' || arg[0] == '[') {
		var v interface{}
		if err := json.Unmarshal([]byte(arg), &v); err == nil {
			b, _ := json.Marshal(v)
			return string(b)
		}
	}
	return arg
}

func parseQueryParams(rawURL string) []models.QueryParam {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	var out []models.QueryParam
	for key, values := range u.Query() {
		for _, v := range values {
			out = append(out, models.QueryParam{
				Key:         key,
				Value:       v,
				Enabled:     true,
				Description: "",
			})
		}
	}
	return out
}
