package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ApiClient/internal/models"
)

func TestExecuteBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"files":[{"name":"a.txt"}]}`))
	}))
	defer server.Close()

	req := models.Request{
		ID:     "test",
		Type:   "http",
		Name:   "test",
		URL:    server.URL + "/list",
		Method: "GET",
	}

	resp, err := NewHttpExecutor().Execute(context.Background(), models.HttpRequest{Request: req, ResolvedURL: req.URL})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(resp.Body) == 0 {
		t.Fatal("expected non-empty body")
	}
	if resp.Body != `{"files":[{"name":"a.txt"}]}` {
		t.Fatalf("unexpected body: %s", resp.Body)
	}
	if resp.Status != "OK" {
		t.Fatalf("expected status OK, got %s", resp.Status)
	}
}
