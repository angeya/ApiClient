package hooks

import (
	"context"
	"testing"
)

// TestExecutePre_LogOutput 验证 pm.log() 输出能被正确收集
func TestExecutePre_LogOutput(t *testing.T) {
	engine := NewScriptEngine()
	req := &ScriptRequest{
		URL:    "http://example.com",
		Method: "GET",
	}

	script := `pm.log(123); pm.log("hello", "world");`

	globalVars := map[string]string{}
	envVars := map[string]string{}

	_, _, logs, modifiedReq, err := engine.ExecutePre(context.Background(), script, req, globalVars, envVars)
	if err != nil {
		t.Fatalf("ExecutePre failed: %v", err)
	}
	if modifiedReq == nil {
		t.Fatal("modifiedReq should not be nil on success")
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d: %v", len(logs), logs)
	}
	if logs[0] != "123" {
		t.Errorf("expected first log '123', got %q", logs[0])
	}
	if logs[1] != "hello world" {
		t.Errorf("expected second log 'hello world', got %q", logs[1])
	}
}

// TestExecutePre_SetGlobalAndEnv 验证变量设置功能
func TestExecutePre_SetGlobalAndEnv(t *testing.T) {
	engine := NewScriptEngine()
	req := &ScriptRequest{
		URL:    "http://example.com",
		Method: "GET",
	}

	script := `
		pm.setGlobal("token", "abc123");
		pm.setEnv("host", "api.example.com");
		pm.log(pm.getGlobal("token"));
		pm.log(pm.getEnv("host"));
	`

	globalVars := map[string]string{}
	envVars := map[string]string{}

	newGlobal, newEnv, logs, _, err := engine.ExecutePre(context.Background(), script, req, globalVars, envVars)
	if err != nil {
		t.Fatalf("ExecutePre failed: %v", err)
	}
	if newGlobal["token"] != "abc123" {
		t.Errorf("expected global token 'abc123', got %q", newGlobal["token"])
	}
	if newEnv["host"] != "api.example.com" {
		t.Errorf("expected env host 'api.example.com', got %q", newEnv["host"])
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d: %v", len(logs), logs)
	}
	if logs[0] != "abc123" {
		t.Errorf("expected first log 'abc123', got %q", logs[0])
	}
	if logs[1] != "api.example.com" {
		t.Errorf("expected second log 'api.example.com', got %q", logs[1])
	}
}

// TestExecutePre_EmptyScript 验证空脚本不会产生日志
func TestExecutePre_EmptyScript(t *testing.T) {
	engine := NewScriptEngine()
	req := &ScriptRequest{URL: "http://example.com", Method: "GET"}

	_, _, logs, _, err := engine.ExecutePre(context.Background(), "", req, nil, nil)
	if err != nil {
		t.Fatalf("ExecutePre with empty script should not error: %v", err)
	}
	if logs != nil {
		t.Errorf("expected nil logs for empty script, got %v", logs)
	}
}

// TestExecutePost_LogObject 验证 pm.log 输出对象时按 JSON 序列化
func TestExecutePost_LogObject(t *testing.T) {
	engine := NewScriptEngine()
	resp := &ScriptResponse{
		StatusCode: 200,
		Status:     "OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"id":1}`,
		TimeMs:     42,
	}

	script := `pm.log(pm.response);`

	_, _, logs, err := engine.ExecutePost(context.Background(), script, resp, nil, nil)
	if err != nil {
		t.Fatalf("ExecutePost failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d: %v", len(logs), logs)
	}
	if logs[0] == "[object Object]" {
		t.Errorf("expected JSON log, got [object Object]")
	}
	if logs[0] == "" {
		t.Errorf("expected non-empty JSON log")
	}
}

// TestExecutePost_ResponseAccessors 验证 pm.response.body 与 pm.response.json() 可正常访问
func TestExecutePost_ResponseAccessors(t *testing.T) {
	engine := NewScriptEngine()
	resp := &ScriptResponse{
		StatusCode: 200,
		Status:     "OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"id":1,"name":"test"}`,
		TimeMs:     42,
	}

	script := `
		pm.log(pm.response.body);
		pm.log(pm.response.statusCode);
		const json = pm.response.json();
		pm.log(json.name);
	`

	_, _, logs, err := engine.ExecutePost(context.Background(), script, resp, nil, nil)
	if err != nil {
		t.Fatalf("ExecutePost failed: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d: %v", len(logs), logs)
	}
	if logs[0] != `{"id":1,"name":"test"}` {
		t.Errorf("expected body log, got %q", logs[0])
	}
	if logs[1] != "200" {
		t.Errorf("expected statusCode 200, got %q", logs[1])
	}
	if logs[2] != "test" {
		t.Errorf("expected json.name 'test', got %q", logs[2])
	}
}

// TestExecutePre_ScriptError 验证脚本错误时仍返回已收集的日志
func TestExecutePre_ScriptError(t *testing.T) {
	engine := NewScriptEngine()
	req := &ScriptRequest{URL: "http://example.com", Method: "GET"}

	script := `
		pm.log("before error");
		throw new Error("test error");
		pm.log("after error");
	`

	_, _, logs, modifiedReq, err := engine.ExecutePre(context.Background(), script, req, nil, nil)
	if err == nil {
		t.Fatal("expected error for script with throw")
	}
	if modifiedReq != nil {
		t.Error("expected nil modifiedReq on error")
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log (before error), got %d: %v", len(logs), logs)
	}
	if logs[0] != "before error" {
		t.Errorf("expected log 'before error', got %q", logs[0])
	}
}
