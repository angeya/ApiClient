package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// ScriptEngine executes pre/post request scripts.
type ScriptEngine struct {
	maxDuration time.Duration
}

// NewScriptEngine creates a script engine.
func NewScriptEngine() *ScriptEngine {
	return &ScriptEngine{maxDuration: 5 * time.Second}
}

// ScriptRequest exposes request fields to scripts.
type ScriptRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// ScriptResponse exposes response fields to scripts.
type ScriptResponse struct {
	StatusCode int                 `json:"statusCode"`
	Status     string              `json:"status"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	TimeMs     int64               `json:"timeMs"`
}

// ExecutePre runs a pre-request script.
func (e *ScriptEngine) ExecutePre(ctx context.Context, script string, req *ScriptRequest, globalVars, envVars map[string]string) (map[string]string, map[string]string, []string, *ScriptRequest, error) {
	if script == "" {
		return cloneMap(globalVars), cloneMap(envVars), nil, req, nil
	}
	vm := goja.New()
	pm := newPmObject(vm, req, nil, cloneMap(globalVars), cloneMap(envVars))
	pm.install(vm)
	if err := e.runWithTimeout(ctx, vm, script); err != nil {
		return pm.globalVars, pm.envVars, pm.logs, nil, err
	}
	return pm.globalVars, pm.envVars, pm.logs, req, nil
}

// ExecutePost runs a post-response script.
func (e *ScriptEngine) ExecutePost(ctx context.Context, script string, resp *ScriptResponse, globalVars, envVars map[string]string) (map[string]string, map[string]string, []string, error) {
	if script == "" {
		return cloneMap(globalVars), cloneMap(envVars), nil, nil
	}
	vm := goja.New()
	pm := newPmObject(vm, nil, resp, cloneMap(globalVars), cloneMap(envVars))
	pm.install(vm)
	if err := e.runWithTimeout(ctx, vm, script); err != nil {
		return pm.globalVars, pm.envVars, pm.logs, err
	}
	return pm.globalVars, pm.envVars, pm.logs, nil
}

func (e *ScriptEngine) runWithTimeout(ctx context.Context, vm *goja.Runtime, script string) error {
	ctx, cancel := context.WithTimeout(ctx, e.maxDuration)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := vm.RunString(script)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("script execution timeout")
	}
}

type pmObject struct {
	vm         *goja.Runtime
	request    *ScriptRequest
	response   *ScriptResponse
	globalVars map[string]string
	envVars    map[string]string
	logs       []string
}

func newPmObject(vm *goja.Runtime, req *ScriptRequest, resp *ScriptResponse, globalVars, envVars map[string]string) *pmObject {
	return &pmObject{
		vm:         vm,
		request:    req,
		response:   resp,
		globalVars: globalVars,
		envVars:    envVars,
		logs:       []string{},
	}
}

func (p *pmObject) install(vm *goja.Runtime) {
	obj := vm.NewObject()

	// pm.log - use goja native callable
	_ = obj.Set("log", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			parts = append(parts, p.valueToString(arg))
		}
		p.logs = append(p.logs, strings.Join(parts, " "))
		return goja.Undefined()
	})

	// pm.setGlobal
	_ = obj.Set("setGlobal", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 2 {
			p.globalVars[call.Arguments[0].String()] = call.Arguments[1].String()
		}
		return goja.Undefined()
	})

	// pm.getGlobal
	_ = obj.Set("getGlobal", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 1 {
			if v, ok := p.globalVars[call.Arguments[0].String()]; ok {
				return p.vm.ToValue(v)
			}
		}
		return goja.Undefined()
	})

	// pm.setEnv
	_ = obj.Set("setEnv", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 2 {
			p.envVars[call.Arguments[0].String()] = call.Arguments[1].String()
		}
		return goja.Undefined()
	})

	// pm.getEnv
	_ = obj.Set("getEnv", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 1 {
			if v, ok := p.envVars[call.Arguments[0].String()]; ok {
				return p.vm.ToValue(v)
			}
		}
		return goja.Undefined()
	})

	// pm.responseJSON
	_ = obj.Set("responseJSON", func(call goja.FunctionCall) goja.Value {
		if p.response == nil {
			return goja.Undefined()
		}
		var v interface{}
		if err := json.Unmarshal([]byte(p.response.Body), &v); err != nil {
			return goja.Undefined()
		}
		return p.vm.ToValue(v)
	})

	// pm.request
	if p.request != nil {
		_ = obj.Set("request", p.vm.ToValue(p.request))
	}

	// pm.response - expose as a proper JS object with body/status/headers and a json() helper.
	if p.response != nil {
		respObj := vm.NewObject()
		_ = respObj.Set("statusCode", p.response.StatusCode)
		_ = respObj.Set("status", p.response.Status)
		_ = respObj.Set("headers", p.response.Headers)
		_ = respObj.Set("body", p.response.Body)
		_ = respObj.Set("timeMs", p.response.TimeMs)
		_ = respObj.Set("json", func(call goja.FunctionCall) goja.Value {
			var v interface{}
			if err := json.Unmarshal([]byte(p.response.Body), &v); err != nil {
				return goja.Undefined()
			}
			return p.vm.ToValue(v)
		})
		_ = obj.Set("response", respObj)
	}

	_ = vm.Set("pm", obj)
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// valueToString converts a goja value to a log string.
// Primitive types keep their native string representation, while objects,
// arrays and structs are serialized as JSON so that pm.log(pm.response)
// prints the actual response data instead of [object Object].
func (p *pmObject) valueToString(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return v.String()
	}
	// Keep strings as-is to avoid wrapping them in quotes.
	if _, ok := v.Export().(string); ok {
		return v.String()
	}
	// Use JSON.stringify(value, null, 2) for objects/arrays so functions are
	// skipped and the output is readable. Fallback to v.String() on failure.
	if s, err := p.jsonStringify(v); err == nil {
		return s
	}
	return v.String()
}

// jsonStringify returns a pretty-printed JSON representation of a goja value.
// It uses the runtime's built-in JSON.stringify, which ignores functions and
// handles native JS objects correctly.
func (p *pmObject) jsonStringify(v goja.Value) (string, error) {
	jsonObj := p.vm.Get("JSON")
	if goja.IsUndefined(jsonObj) || goja.IsNull(jsonObj) {
		return "", fmt.Errorf("JSON not available")
	}
	stringify := jsonObj.ToObject(p.vm).Get("stringify")
	if callable, ok := goja.AssertFunction(stringify); ok {
		result, err := callable(goja.Undefined(), v, goja.Null(), p.vm.ToValue("  "))
		if err != nil {
			return "", err
		}
		return result.String(), nil
	}
	return "", fmt.Errorf("JSON.stringify is not callable")
}
