package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ApiClient/internal/models"
)

// HttpExecutor handles HTTP request execution.
type HttpExecutor struct {
	client *http.Client
}

// NewHttpExecutor creates an HTTP executor.
func NewHttpExecutor() *HttpExecutor {
	return &HttpExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// Execute sends an HTTP request and returns the response.
func (e *HttpExecutor) Execute(ctx context.Context, req models.HttpRequest) (models.HttpResponse, error) {
	var resp models.HttpResponse
	u, err := buildURL(req.ResolvedURL, req.Query)
	if err != nil {
		return resp, err
	}

	var bodyReader io.Reader
	if len(req.ResolvedBody) > 0 {
		bodyReader = bytes.NewReader(req.ResolvedBody)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u, bodyReader)
	if err != nil {
		return resp, err
	}

	for _, h := range req.ResolvedHeaders {
		httpReq.Header.Set(h.Key, h.Value)
	}

	start := time.Now()
	httpResp, err := e.client.Do(httpReq)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()
	serverTime := time.Since(start)

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 10*1024*1024))
	if err != nil {
		return resp, err
	}
	totalTime := time.Since(start)

	resp.StatusCode = httpResp.StatusCode
	resp.Status = statusText(httpResp.Status)
	resp.Headers = httpResp.Header
	resp.Body = string(body)
	resp.TimeMs = totalTime.Milliseconds()
	resp.ServerTimeMs = serverTime.Milliseconds()
	resp.TransferTimeMs = (totalTime - serverTime).Milliseconds()
	resp.SizeBytes = int64(len(body))
	resp.ContentType = httpResp.Header.Get("Content-Type")
	for _, c := range httpResp.Cookies() {
		resp.Cookies = append(resp.Cookies, c.String())
	}
	return resp, nil
}

func statusText(status string) string {
	parts := strings.SplitN(status, " ", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return status
}

func buildURL(raw string, query []models.QueryParam) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for _, p := range query {
		if !p.Enabled || p.Key == "" {
			continue
		}
		q.Set(p.Key, p.Value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// BuildBody builds the request body based on body type.
func BuildBody(req models.Request) ([]byte, string, error) {
	switch req.BodyType {
	case "none", "":
		return nil, "", nil
	case "json", "text":
		return []byte(req.Body), ContentTypeFor(req.BodyType), nil
	case "x-www-form-urlencoded":
		data := url.Values{}
		for _, f := range req.FormData {
			if !f.Enabled || f.Type != "text" {
				continue
			}
			// 使用 Add 保留同名参数的多个值，更符合表单语义
			data.Add(f.Key, f.Value)
		}
		return []byte(data.Encode()), "application/x-www-form-urlencoded", nil
	case "form-data":
		return nil, "multipart/form-data", nil
	default:
		return []byte(req.Body), "text/plain", nil
	}
}

func ContentTypeFor(bodyType string) string {
	switch bodyType {
	case "json":
		return "application/json"
	case "text":
		return "text/plain"
	default:
		return "text/plain"
	}
}

// ValidateJSONBody validates JSON body content.
func ValidateJSONBody(body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

// BuildMultipartBody builds multipart body for form-data requests.
func BuildMultipartBody(req models.Request) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, f := range req.FormData {
		if !f.Enabled {
			continue
		}
		if f.Type == "text" {
			_ = writer.WriteField(f.Key, f.Value)
			continue
		}
		// 文件字段：f.Value 必须是可访问的文件路径
		filePath := strings.TrimSpace(f.Value)
		if filePath == "" {
			_ = writer.Close()
			return nil, "", fmt.Errorf("字段 %s 的文件路径为空", f.Key)
		}
		file, err := os.Open(filePath)
		if err != nil {
			_ = writer.Close()
			return nil, "", fmt.Errorf("字段 %s 无法打开文件 %s: %w", f.Key, filePath, err)
		}
		part, err := writer.CreateFormFile(f.Key, filepath.Base(filePath))
		if err != nil {
			file.Close()
			_ = writer.Close()
			return nil, "", err
		}
		_, err = io.Copy(part, file)
		file.Close()
		if err != nil {
			_ = writer.Close()
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}
