package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// mockTransport implements Transport for testing.
type mockTransport struct {
	handler func(req *Request) (*Response, error)
	closed  bool
}

func (m *mockTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	return m.handler(req)
}

func (m *mockTransport) SendNotification(ctx context.Context, notif *Request) error {
	if m.handler != nil {
		_, err := m.handler(notif)
		return err
	}
	return nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

// reqID safely extracts the request ID, returning 0 for notifications.
func reqID(req *Request) int64 {
	if req.ID != nil {
		return reqID(req)
	}
	return 0
}

// helper to create a Response with a JSON result.
func jsonResponse(id int64, result any) *Response {
	raw, _ := json.Marshal(result)
	return &Response{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(raw),
	}
}

// helper to create an error Response.
func errorResponse(id int64, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      &id,
		Error:   &RPCError{Code: code, Message: message},
	}
}

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		handler func(req *Request) (*Response, error)
		wantErr bool
	}{
		{
			name: "successful handshake",
			handler: func(req *Request) (*Response, error) {
				switch req.Method {
				case "initialize":
					return jsonResponse(reqID(req), map[string]any{
						"protocolVersion": "2024-11-05",
						"serverInfo":      map[string]any{"name": "test-server", "version": "1.0"},
						"capabilities":    map[string]any{},
					}), nil
				case "notifications/initialized":
					return &Response{JSONRPC: "2.0"}, nil
				default:
					return nil, errors.New("unexpected method: " + req.Method)
				}
			},
			wantErr: false,
		},
		{
			name: "server error response",
			handler: func(req *Request) (*Response, error) {
				if req.Method == "initialize" {
					return errorResponse(reqID(req), -32600, "unsupported protocol version"), nil
				}
				return &Response{JSONRPC: "2.0"}, nil
			},
			wantErr: true,
		},
		{
			name: "transport error",
			handler: func(req *Request) (*Response, error) {
				return nil, errors.New("connection refused")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			err := c.Initialize(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListTools(t *testing.T) {
	tests := []struct {
		name      string
		handler   func(req *Request) (*Response, error)
		wantCount int
		wantErr   bool
	}{
		{
			name: "returns tools",
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), listToolsResult{
					Tools: []Tool{
						{
							Name:        "read_file",
							Description: "Read a file",
							InputSchema: InputSchema{
								Type: "object",
								Properties: map[string]PropertySchema{
									"path": {Type: "string", Description: "File path"},
								},
								Required: []string{"path"},
							},
						},
						{
							Name:        "write_file",
							Description: "Write a file",
							InputSchema: InputSchema{
								Type: "object",
								Properties: map[string]PropertySchema{
									"path":    {Type: "string", Description: "File path"},
									"content": {Type: "string", Description: "Content"},
								},
								Required: []string{"path", "content"},
							},
						},
					},
				}), nil
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty list",
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), listToolsResult{Tools: []Tool{}}), nil
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "rpc error",
			handler: func(req *Request) (*Response, error) {
				return errorResponse(reqID(req), -32601, "method not found"), nil
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			tools, err := c.ListTools(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListTools() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tools) != tt.wantCount {
				t.Errorf("ListTools() returned %d tools, want %d", len(tools), tt.wantCount)
			}
		})
	}
}

func TestCallTool(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		args        map[string]any
		handler     func(req *Request) (*Response, error)
		wantText    string
		wantToolErr bool
		wantErr     bool
	}{
		{
			name:     "successful call with text content",
			toolName: "read_file",
			args:     map[string]any{"path": "/tmp/test.txt"},
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), CallResult{
					Content: []Content{{Type: "text", Text: "hello world"}},
					IsError: false,
				}), nil
			},
			wantText:    "hello world",
			wantToolErr: false,
			wantErr:     false,
		},
		{
			name:     "tool error (isError=true)",
			toolName: "read_file",
			args:     map[string]any{"path": "/nonexistent"},
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), CallResult{
					Content: []Content{{Type: "text", Text: "file not found"}},
					IsError: true,
				}), nil
			},
			wantText:    "file not found",
			wantToolErr: true,
			wantErr:     true,
		},
		{
			name:     "rpc error",
			toolName: "unknown",
			args:     map[string]any{},
			handler: func(req *Request) (*Response, error) {
				return errorResponse(reqID(req), -32601, "method not found"), nil
			},
			wantToolErr: false,
			wantErr:     true,
		},
		{
			name:     "transport error",
			toolName: "read_file",
			args:     map[string]any{},
			handler: func(req *Request) (*Response, error) {
				return nil, errors.New("broken pipe")
			},
			wantToolErr: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			result, err := c.CallTool(context.Background(), tt.toolName, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("CallTool() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantToolErr {
				var toolErr *ToolError
				if !errors.As(err, &toolErr) {
					t.Errorf("CallTool() expected ToolError, got %T: %v", err, err)
				}
			}

			if tt.wantText != "" && result != nil && len(result.Content) > 0 {
				if result.Content[0].Text != tt.wantText {
					t.Errorf("CallTool() text = %q, want %q", result.Content[0].Text, tt.wantText)
				}
			}
		})
	}
}

func TestCallToolRequestParams(t *testing.T) {
	var captured *Request
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			captured = req
			return jsonResponse(reqID(req), CallResult{
				Content: []Content{{Type: "text", Text: "ok"}},
			}), nil
		},
	}

	c := NewClient(mt)
	_, _ = c.CallTool(context.Background(), "my_tool", map[string]any{"key": "value"})

	if captured == nil {
		t.Fatal("no request was sent")
	}
	if captured.Method != "tools/call" {
		t.Errorf("method = %q, want %q", captured.Method, "tools/call")
	}

	params, ok := captured.Params.(map[string]any)
	if !ok {
		t.Fatalf("params type = %T, want map[string]any", captured.Params)
	}
	if params["name"] != "my_tool" {
		t.Errorf("params.name = %v, want %q", params["name"], "my_tool")
	}
}

func TestClientClose(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			return &Response{JSONRPC: "2.0"}, nil
		},
	}

	c := NewClient(mt)
	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !mt.closed {
		t.Error("Close() did not close transport")
	}
}

func TestToolErrorFormat(t *testing.T) {
	e := &ToolError{Name: "read_file", Message: "not found", Code: 0}
	want := `tool "read_file" error: not found`
	if e.Error() != want {
		t.Errorf("ToolError.Error() = %q, want %q", e.Error(), want)
	}

	e2 := &ToolError{Name: "read_file", Message: "not found", Code: 404}
	want2 := `tool "read_file" error (code 404): not found`
	if e2.Error() != want2 {
		t.Errorf("ToolError.Error() = %q, want %q", e2.Error(), want2)
	}
}

func TestRPCErrorFormat(t *testing.T) {
	e := &RPCError{Code: -32601, Message: "method not found"}
	want := "rpc error (code -32601): method not found"
	if e.Error() != want {
		t.Errorf("RPCError.Error() = %q, want %q", e.Error(), want)
	}
}
