package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// Compile-time interface checks.
var (
	_ Transport         = (*HTTPTransport)(nil)
	_ NotificationAware = (*HTTPTransport)(nil)
)

// HTTPTransport implements the MCP Streamable HTTP transport (2025-11-25 spec).
// It sends JSON-RPC requests via POST and handles both application/json and
// text/event-stream response content types.
type HTTPTransport struct {
	baseURL   string
	sessionID string
	client    *http.Client
	headers   map[string]string

	nextID atomic.Int64

	done         chan struct{}
	closeOnce    sync.Once
	initialized  bool
	notifHandler NotificationHandler
}

// NewHTTPTransport creates a new Streamable HTTP transport.
func NewHTTPTransport(baseURL string, headers map[string]string) *HTTPTransport {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &HTTPTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
		headers: headers,
		done:    make(chan struct{}),
	}
}

// SetNotificationHandler sets a callback for server-initiated notifications.
func (t *HTTPTransport) SetNotificationHandler(handler NotificationHandler) {
	t.notifHandler = handler
}

// Send sends a JSON-RPC request via POST and waits for the response.
func (t *HTTPTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	resp, err := t.doSend(ctx, req, true)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *HTTPTransport) doSend(ctx context.Context, req *Request, allowRetry bool) (*Response, error) {
	id := t.nextID.Add(1)
	req.ID = &id
	req.JSONRPC = "2.0"

	select {
	case <-t.done:
		return nil, ErrTransportClosed
	default:
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mcp: http marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp: http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if t.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	if t.initialized {
		httpReq.Header.Set("MCP-Protocol-Version", "2025-11-25")
	}
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp: http do: %w", err)
	}
	defer httpResp.Body.Close()

	// Handle session expiration: 404 means re-initialize.
	if httpResp.StatusCode == http.StatusNotFound && t.sessionID != "" && allowRetry {
		t.sessionID = ""
		t.initialized = false
		return t.doSend(ctx, req, false)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("mcp: http status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Capture session ID from response.
	if sid := httpResp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	// Mark as initialized after the init request succeeds.
	if req.Method == "initialize" {
		t.initialized = true
	}

	ct := httpResp.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "text/event-stream") {
		return t.readSSEResponse(httpResp.Body, id)
	}

	// Default: application/json
	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("mcp: http decode: %w", err)
	}
	return &resp, nil
}

// SendNotification sends a JSON-RPC notification via POST, expecting 202 Accepted.
func (t *HTTPTransport) SendNotification(ctx context.Context, notif *Request) error {
	notif.ID = nil
	notif.JSONRPC = "2.0"

	select {
	case <-t.done:
		return ErrTransportClosed
	default:
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("mcp: http marshal notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mcp: http notification request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	if t.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	if t.initialized {
		httpReq.Header.Set("MCP-Protocol-Version", "2025-11-25")
	}
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("mcp: http notification: %w", err)
	}
	httpResp.Body.Close()

	// Accept 2xx responses — server may return 202 Accepted, 200 OK, or 204 No Content.
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("mcp: http notification status %d", httpResp.StatusCode)
	}

	return nil
}

// Close sends a DELETE to terminate the session and cleans up.
func (t *HTTPTransport) Close() error {
	var closeErr error
	t.closeOnce.Do(func() {
		close(t.done)

		if t.sessionID != "" {
			req, err := http.NewRequest(http.MethodDelete, t.baseURL, nil)
			if err == nil {
				req.Header.Set("Mcp-Session-Id", t.sessionID)
				for k, v := range t.headers {
					req.Header.Set(k, v)
				}
				resp, err := t.client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}

	})
	return closeErr
}

// readSSEResponse reads an SSE stream until we find the response matching our request ID.
func (t *HTTPTransport) readSSEResponse(r io.Reader, requestID int64) (*Response, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Blank line: dispatch accumulated event.
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				resp, done := t.handleSSEEvent(eventType, data, requestID)
				if done {
					return resp, nil
				}
			}
			eventType = ""
			dataLines = nil
			continue
		}

		if val, ok := strings.CutPrefix(line, "event:"); ok {
			eventType = strings.TrimSpace(val)
		} else if val, ok := strings.CutPrefix(line, "data:"); ok {
			dataLines = append(dataLines, val)
		}
		// Ignore id:, retry:, comments (:)
	}

	// If we reach EOF without finding our response, flush any buffered event.
	if len(dataLines) > 0 {
		data := strings.Join(dataLines, "\n")
		resp, done := t.handleSSEEvent(eventType, data, requestID)
		if done {
			return resp, nil
		}
	}

	return nil, fmt.Errorf("mcp: http SSE stream ended without response for request %d", requestID)
}

func (t *HTTPTransport) handleSSEEvent(_, data string, requestID int64) (*Response, bool) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, false
	}

	// Try to parse as a JSON-RPC response.
	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, false
	}

	// If it has an ID matching our request, return it.
	if resp.ID != nil && *resp.ID == requestID {
		return &resp, true
	}

	// If it has no ID, it's a notification.
	if resp.ID == nil {
		var notif Notification
		if err := json.Unmarshal([]byte(data), &notif); err == nil && notif.Method != "" {
			if t.notifHandler != nil {
				t.notifHandler(notif.Method, notif.Params)
			}
		}
	}

	return nil, false
}
