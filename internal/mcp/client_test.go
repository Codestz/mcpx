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
		return *req.ID
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

func TestInitializeParsesCaps(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			switch req.Method {
			case "initialize":
				return jsonResponse(reqID(req), map[string]any{
					"protocolVersion": "2025-11-25",
					"serverInfo":      map[string]any{"name": "cap-server", "version": "2.0"},
					"capabilities": map[string]any{
						"tools": map[string]any{"listChanged": true},
					},
				}), nil
			case "notifications/initialized":
				return &Response{JSONRPC: "2.0"}, nil
			default:
				return nil, errors.New("unexpected method")
			}
		},
	}

	c := NewClient(mt)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	info := c.ServerInfo()
	if info.Name != "cap-server" || info.Version != "2.0" {
		t.Errorf("ServerInfo = %+v, want name=cap-server version=2.0", info)
	}

	caps := c.ServerCapabilities()
	if caps.Tools == nil || !caps.Tools.ListChanged {
		t.Errorf("ServerCapabilities.Tools.ListChanged = false, want true")
	}
}

func TestListToolsPaginated(t *testing.T) {
	callCount := 0
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			callCount++

			// Check if cursor was sent.
			var cursor string
			if req.Params != nil {
				if params, ok := req.Params.(map[string]any); ok {
					if c, ok := params["cursor"].(string); ok {
						cursor = c
					}
				}
			}

			switch cursor {
			case "":
				// First page.
				next := "page2"
				return jsonResponse(reqID(req), listToolsResult{
					Tools:      []Tool{{Name: "tool_a", Description: "A"}},
					NextCursor: &next,
				}), nil
			case "page2":
				// Second page.
				next := "page3"
				return jsonResponse(reqID(req), listToolsResult{
					Tools:      []Tool{{Name: "tool_b", Description: "B"}},
					NextCursor: &next,
				}), nil
			case "page3":
				// Last page (no cursor).
				return jsonResponse(reqID(req), listToolsResult{
					Tools: []Tool{{Name: "tool_c", Description: "C"}},
				}), nil
			default:
				return nil, errors.New("unexpected cursor: " + cursor)
			}
		},
	}

	c := NewClient(mt)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 3 {
		t.Fatalf("ListTools() returned %d tools, want 3", len(tools))
	}

	names := []string{tools[0].Name, tools[1].Name, tools[2].Name}
	want := []string{"tool_a", "tool_b", "tool_c"}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("tool[%d].Name = %q, want %q", i, n, want[i])
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 requests (3 pages), got %d", callCount)
	}
}

func TestListPrompts(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			return jsonResponse(reqID(req), listPromptsResult{
				Prompts: []Prompt{
					{Name: "code_review", Description: "Review code for issues"},
					{Name: "summarize", Description: "Summarize text"},
				},
			}), nil
		},
	}

	c := NewClient(mt)
	prompts, err := c.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("ListPrompts() returned %d prompts, want 2", len(prompts))
	}
	if prompts[0].Name != "code_review" {
		t.Errorf("prompts[0].Name = %q, want %q", prompts[0].Name, "code_review")
	}
	if prompts[1].Name != "summarize" {
		t.Errorf("prompts[1].Name = %q, want %q", prompts[1].Name, "summarize")
	}
}

func TestListPromptsPaginated(t *testing.T) {
	callCount := 0
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			callCount++

			var cursor string
			if req.Params != nil {
				if params, ok := req.Params.(map[string]any); ok {
					if c, ok := params["cursor"].(string); ok {
						cursor = c
					}
				}
			}

			switch cursor {
			case "":
				next := "page2"
				return jsonResponse(reqID(req), listPromptsResult{
					Prompts:    []Prompt{{Name: "prompt_a", Description: "A"}},
					NextCursor: &next,
				}), nil
			case "page2":
				next := "page3"
				return jsonResponse(reqID(req), listPromptsResult{
					Prompts:    []Prompt{{Name: "prompt_b", Description: "B"}},
					NextCursor: &next,
				}), nil
			case "page3":
				return jsonResponse(reqID(req), listPromptsResult{
					Prompts: []Prompt{{Name: "prompt_c", Description: "C"}},
				}), nil
			default:
				return nil, errors.New("unexpected cursor: " + cursor)
			}
		},
	}

	c := NewClient(mt)
	prompts, err := c.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}
	if len(prompts) != 3 {
		t.Fatalf("ListPrompts() returned %d prompts, want 3", len(prompts))
	}
	if callCount != 3 {
		t.Errorf("expected 3 requests (3 pages), got %d", callCount)
	}
}

func TestGetPrompt(t *testing.T) {
	tests := []struct {
		name     string
		handler  func(req *Request) (*Response, error)
		wantMsgs int
		wantErr  bool
	}{
		{
			name: "successful get with args",
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), PromptResult{
					Messages: []PromptMessage{
						{Role: "user", Content: Content{Type: "text", Text: "Review this code"}},
						{Role: "assistant", Content: Content{Type: "text", Text: "LGTM"}},
					},
				}), nil
			},
			wantMsgs: 2,
			wantErr:  false,
		},
		{
			name: "rpc error",
			handler: func(req *Request) (*Response, error) {
				return errorResponse(reqID(req), -32601, "method not found"), nil
			},
			wantMsgs: 0,
			wantErr:  true,
		},
		{
			name: "transport error",
			handler: func(req *Request) (*Response, error) {
				return nil, errors.New("connection lost")
			},
			wantMsgs: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			result, err := c.GetPrompt(context.Background(), "test_prompt", map[string]string{"lang": "go"})
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result != nil && len(result.Messages) != tt.wantMsgs {
				t.Errorf("GetPrompt() returned %d messages, want %d", len(result.Messages), tt.wantMsgs)
			}
		})
	}
}

func TestListResources(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			return jsonResponse(reqID(req), listResourcesResult{
				Resources: []Resource{
					{URI: "file:///src/main.go", Name: "main.go", Description: "Entry point"},
					{URI: "file:///src/lib.go", Name: "lib.go"},
				},
			}), nil
		},
	}

	c := NewClient(mt)
	resources, err := c.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("ListResources() returned %d resources, want 2", len(resources))
	}
	if resources[0].URI != "file:///src/main.go" {
		t.Errorf("resources[0].URI = %q, want %q", resources[0].URI, "file:///src/main.go")
	}
}

func TestListResourcesPaginated(t *testing.T) {
	callCount := 0
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			callCount++

			var cursor string
			if req.Params != nil {
				if params, ok := req.Params.(map[string]any); ok {
					if c, ok := params["cursor"].(string); ok {
						cursor = c
					}
				}
			}

			switch cursor {
			case "":
				next := "page2"
				return jsonResponse(reqID(req), listResourcesResult{
					Resources:  []Resource{{URI: "file:///a", Name: "a"}},
					NextCursor: &next,
				}), nil
			case "page2":
				next := "page3"
				return jsonResponse(reqID(req), listResourcesResult{
					Resources:  []Resource{{URI: "file:///b", Name: "b"}},
					NextCursor: &next,
				}), nil
			case "page3":
				return jsonResponse(reqID(req), listResourcesResult{
					Resources: []Resource{{URI: "file:///c", Name: "c"}},
				}), nil
			default:
				return nil, errors.New("unexpected cursor: " + cursor)
			}
		},
	}

	c := NewClient(mt)
	resources, err := c.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if len(resources) != 3 {
		t.Fatalf("ListResources() returned %d resources, want 3", len(resources))
	}
	if callCount != 3 {
		t.Errorf("expected 3 requests (3 pages), got %d", callCount)
	}
}

func TestReadResource(t *testing.T) {
	tests := []struct {
		name    string
		handler func(req *Request) (*Response, error)
		wantErr bool
	}{
		{
			name: "successful read with text",
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), ResourceResult{
					Contents: []ResourceContent{
						{URI: "file:///test.txt", Text: "hello world", MimeType: "text/plain"},
					},
				}), nil
			},
			wantErr: false,
		},
		{
			name: "successful read with blob",
			handler: func(req *Request) (*Response, error) {
				return jsonResponse(reqID(req), ResourceResult{
					Contents: []ResourceContent{
						{URI: "file:///img.png", Blob: "aGVsbG8=", MimeType: "image/png"},
					},
				}), nil
			},
			wantErr: false,
		},
		{
			name: "rpc error",
			handler: func(req *Request) (*Response, error) {
				return errorResponse(reqID(req), -32601, "method not found"), nil
			},
			wantErr: true,
		},
		{
			name: "transport error",
			handler: func(req *Request) (*Response, error) {
				return nil, errors.New("broken pipe")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			result, err := c.ReadResource(context.Background(), "file:///test.txt")
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadResource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != nil && len(result.Contents) == 0 {
				t.Error("ReadResource() returned empty contents")
			}
		})
	}
}

func TestListResourceTemplates(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			return jsonResponse(reqID(req), listResourceTemplatesResult{
				ResourceTemplates: []ResourceTemplate{
					{URITemplate: "file:///{path}", Name: "files", Description: "Project files"},
					{URITemplate: "db:///{table}/{id}", Name: "records", Description: "Database records"},
				},
			}), nil
		},
	}

	c := NewClient(mt)
	templates, err := c.ListResourceTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListResourceTemplates() error = %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("ListResourceTemplates() returned %d templates, want 2", len(templates))
	}
	if templates[0].URITemplate != "file:///{path}" {
		t.Errorf("templates[0].URITemplate = %q, want %q", templates[0].URITemplate, "file:///{path}")
	}
}

func TestInitializeParsesPromptAndResourceCaps(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *Request) (*Response, error) {
			switch req.Method {
			case "initialize":
				return jsonResponse(reqID(req), map[string]any{
					"protocolVersion": "2025-11-25",
					"serverInfo":      map[string]any{"name": "full-server", "version": "3.0"},
					"capabilities": map[string]any{
						"tools":     map[string]any{"listChanged": true},
						"prompts":   map[string]any{"listChanged": true},
						"resources": map[string]any{"subscribe": true, "listChanged": true},
					},
				}), nil
			case "notifications/initialized":
				return &Response{JSONRPC: "2.0"}, nil
			default:
				return nil, errors.New("unexpected method")
			}
		},
	}

	c := NewClient(mt)
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	caps := c.ServerCapabilities()
	if caps.Prompts == nil {
		t.Fatal("ServerCapabilities.Prompts is nil, want non-nil")
	}
	if !caps.Prompts.ListChanged {
		t.Error("Prompts.ListChanged = false, want true")
	}

	if caps.Resources == nil {
		t.Fatal("ServerCapabilities.Resources is nil, want non-nil")
	}
	if !caps.Resources.Subscribe {
		t.Error("Resources.Subscribe = false, want true")
	}
	if !caps.Resources.ListChanged {
		t.Error("Resources.ListChanged = false, want true")
	}

	if c.ProtocolVersion() != "2025-11-25" {
		t.Errorf("ProtocolVersion() = %q, want %q", c.ProtocolVersion(), "2025-11-25")
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		handler func(req *Request) (*Response, error)
		wantErr bool
	}{
		{
			name: "successful ping",
			handler: func(req *Request) (*Response, error) {
				if req.Method != "ping" {
					return nil, errors.New("expected ping method, got " + req.Method)
				}
				return jsonResponse(reqID(req), map[string]any{}), nil
			},
			wantErr: false,
		},
		{
			name: "ping transport error",
			handler: func(req *Request) (*Response, error) {
				return nil, errors.New("connection lost")
			},
			wantErr: true,
		},
		{
			name: "ping rpc error",
			handler: func(req *Request) (*Response, error) {
				return errorResponse(reqID(req), -32601, "method not found"), nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(&mockTransport{handler: tt.handler})
			err := c.Ping(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
