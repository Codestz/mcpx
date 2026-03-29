package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/mcp"
)

// mockTransport implements mcp.Transport for testing.
type mockTransport struct {
	handler func(req *mcp.Request) (*mcp.Response, error)
}

func (m *mockTransport) Send(ctx context.Context, req *mcp.Request) (*mcp.Response, error) {
	return m.handler(req)
}

func (m *mockTransport) SendNotification(ctx context.Context, notif *mcp.Request) error {
	return nil
}

func (m *mockTransport) Close() error {
	return nil
}

func reqID(req *mcp.Request) int64 {
	if req.ID != nil {
		return *req.ID
	}
	return 0
}

func jsonResponse(id int64, result any) *mcp.Response {
	raw, _ := json.Marshal(result)
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(raw),
	}
}

func TestRunOnConnect_NoHooks(t *testing.T) {
	client := mcp.NewClient(&mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			return nil, errors.New("should not be called")
		},
	})

	err := RunOnConnect(context.Background(), client, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = RunOnConnect(context.Background(), client, "test", []config.LifecycleHook{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunOnConnect_ActivateSuccess(t *testing.T) {
	var capturedTool string
	var capturedArgs map[string]any

	mt := &mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			if req.Method == "tools/call" {
				params, _ := req.Params.(map[string]any)
				capturedTool, _ = params["name"].(string)
				if args, ok := params["arguments"].(map[string]any); ok {
					capturedArgs = args
				}
				return jsonResponse(reqID(req), mcp.CallResult{
					Content: []mcp.Content{{Type: "text", Text: "Project activated"}},
				}), nil
			}
			return jsonResponse(reqID(req), map[string]any{}), nil
		},
	}

	client := mcp.NewClient(mt)

	hooks := []config.LifecycleHook{
		{
			Tool: "activate_project",
			Args: map[string]any{"project": "/tmp/myproject"},
		},
	}

	err := RunOnConnect(context.Background(), client, "serena", hooks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedTool != "activate_project" {
		t.Errorf("tool = %q, want activate_project", capturedTool)
	}
	if capturedArgs["project"] != "/tmp/myproject" {
		t.Errorf("project arg = %v, want /tmp/myproject", capturedArgs["project"])
	}
}

func TestRunOnConnect_ActivateFailure_ToolError(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			if req.Method == "tools/call" {
				return jsonResponse(reqID(req), mcp.CallResult{
					Content: []mcp.Content{{Type: "text", Text: "No project found at /tmp/nonexistent"}},
					IsError: true,
				}), nil
			}
			return jsonResponse(reqID(req), map[string]any{}), nil
		},
	}

	client := mcp.NewClient(mt)

	hooks := []config.LifecycleHook{
		{
			Tool: "activate_project",
			Args: map[string]any{"project": "/tmp/nonexistent"},
		},
	}

	err := RunOnConnect(context.Background(), client, "serena", hooks)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "lifecycle hook") {
		t.Errorf("error should mention lifecycle hook: %s", errMsg)
	}
	if !strings.Contains(errMsg, "activate_project") {
		t.Errorf("error should mention activate_project: %s", errMsg)
	}
	if !strings.Contains(errMsg, "/tmp/nonexistent") {
		t.Errorf("error should mention project path: %s", errMsg)
	}
	if !strings.Contains(errMsg, "onboarding") {
		t.Errorf("error should include onboarding hint: %s", errMsg)
	}
	if !strings.Contains(errMsg, "serena") {
		t.Errorf("error should mention server name: %s", errMsg)
	}
}

func TestRunOnConnect_TransportError(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			if req.Method == "tools/call" {
				return nil, errors.New("connection refused")
			}
			return jsonResponse(reqID(req), map[string]any{}), nil
		},
	}

	client := mcp.NewClient(mt)

	hooks := []config.LifecycleHook{
		{Tool: "activate_project", Args: map[string]any{"project": "/tmp/test"}},
	}

	err := RunOnConnect(context.Background(), client, "serena", hooks)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "lifecycle hook") {
		t.Errorf("error should mention lifecycle hook: %s", err.Error())
	}
}

func TestRunOnConnect_MultipleHooks_StopsOnFailure(t *testing.T) {
	callCount := 0

	mt := &mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			if req.Method == "tools/call" {
				callCount++
				if callCount == 2 {
					return jsonResponse(reqID(req), mcp.CallResult{
						Content: []mcp.Content{{Type: "text", Text: "failed"}},
						IsError: true,
					}), nil
				}
				return jsonResponse(reqID(req), mcp.CallResult{
					Content: []mcp.Content{{Type: "text", Text: "ok"}},
				}), nil
			}
			return jsonResponse(reqID(req), map[string]any{}), nil
		},
	}

	client := mcp.NewClient(mt)

	hooks := []config.LifecycleHook{
		{Tool: "hook_a", Args: map[string]any{}},
		{Tool: "hook_b", Args: map[string]any{}},
		{Tool: "hook_c", Args: map[string]any{}},
	}

	err := RunOnConnect(context.Background(), client, "test", hooks)
	if err == nil {
		t.Fatal("expected error from hook_b")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (stopped at failure), got %d", callCount)
	}
}

func TestRunOnConnect_GenericHookHint(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *mcp.Request) (*mcp.Response, error) {
			if req.Method == "tools/call" {
				return nil, errors.New("tool not supported")
			}
			return jsonResponse(reqID(req), map[string]any{}), nil
		},
	}

	client := mcp.NewClient(mt)

	hooks := []config.LifecycleHook{
		{Tool: "custom_setup", Args: map[string]any{}},
	}

	err := RunOnConnect(context.Background(), client, "test", hooks)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "custom_setup") {
		t.Errorf("error should mention tool name: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "supports tool") {
		t.Errorf("error should include generic hint: %s", err.Error())
	}
}
