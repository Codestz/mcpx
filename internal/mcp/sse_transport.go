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
	_ Transport         = (*SSETransport)(nil)
	_ NotificationAware = (*SSETransport)(nil)
)

// SSETransport implements the legacy MCP HTTP+SSE transport.
// It opens a persistent GET connection for SSE events and sends requests via POST.
type SSETransport struct {
	sseURL  string
	postURL string
	client  *http.Client
	headers map[string]string

	pending map[int64]chan *Response
	mu      sync.Mutex
	nextID  atomic.Int64

	done         chan struct{}
	closeOnce    sync.Once
	sseBody      io.ReadCloser
	scanner      *bufio.Scanner // shared between connect and readLoop
	notifHandler NotificationHandler
}

// NewSSETransport connects to a legacy SSE MCP server.
// It opens the SSE stream and waits for the "endpoint" event to learn the POST URL.
func NewSSETransport(ctx context.Context, sseURL string, headers map[string]string) (*SSETransport, error) {
	if headers == nil {
		headers = make(map[string]string)
	}

	t := &SSETransport{
		sseURL:  sseURL,
		client:  &http.Client{},
		headers: headers,
		pending: make(map[int64]chan *Response),
		done:    make(chan struct{}),
	}

	if err := t.connect(ctx); err != nil {
		return nil, err
	}

	go t.readLoop()

	return t, nil
}

// SetNotificationHandler sets a callback for server-initiated notifications.
func (t *SSETransport) SetNotificationHandler(handler NotificationHandler) {
	t.notifHandler = handler
}

func (t *SSETransport) connect(ctx context.Context) error {
	// Use a background context for the HTTP request so the SSE stream
	// outlives the connect timeout. We use ctx only to bound the
	// endpoint discovery phase below.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, t.sseURL, nil)
	if err != nil {
		return fmt.Errorf("mcp: sse request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("mcp: sse connect: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("mcp: sse status %d", resp.StatusCode)
	}

	t.sseBody = resp.Body

	// Create scanner once — shared between connect and readLoop.
	t.scanner = bufio.NewScanner(resp.Body)
	t.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Read until we get the "endpoint" event.
	// Monitor ctx for timeout during endpoint discovery.
	var eventType string
	var dataLines []string

	for t.scanner.Scan() {
		select {
		case <-ctx.Done():
			resp.Body.Close()
			return fmt.Errorf("mcp: sse endpoint discovery: %w", ctx.Err())
		default:
		}
		line := t.scanner.Text()

		if line == "" {
			if eventType == "endpoint" && len(dataLines) > 0 {
				postURL := strings.TrimSpace(strings.Join(dataLines, ""))
				// Resolve relative URL.
				if strings.HasPrefix(postURL, "/") {
					// Extract base URL from sseURL.
					base := t.sseURL
					if idx := strings.Index(base, "://"); idx >= 0 {
						if slash := strings.Index(base[idx+3:], "/"); slash >= 0 {
							base = base[:idx+3+slash]
						}
					}
					postURL = base + postURL
				}
				t.postURL = postURL
				return nil
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
	}

	if err := t.scanner.Err(); err != nil {
		return fmt.Errorf("mcp: sse read endpoint: %w", err)
	}
	return fmt.Errorf("mcp: sse stream ended before endpoint event")
}

// Send posts a JSON-RPC request and waits for the response on the SSE stream.
func (t *SSETransport) Send(ctx context.Context, req *Request) (*Response, error) {
	id := t.nextID.Add(1)
	req.ID = &id
	req.JSONRPC = "2.0"

	ch := make(chan *Response, 1)

	t.mu.Lock()
	select {
	case <-t.done:
		t.mu.Unlock()
		return nil, ErrTransportClosed
	default:
	}
	t.pending[id] = ch
	t.mu.Unlock()

	body, err := json.Marshal(req)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp: sse marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.postURL, bytes.NewReader(body))
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp: sse post request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp: sse post: %w", err)
	}
	httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp: sse post status %d", httpResp.StatusCode)
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, ErrTransportClosed
		}
		return resp, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	case <-t.done:
		return nil, ErrTransportClosed
	}
}

// SendNotification sends a notification via POST (no response expected).
func (t *SSETransport) SendNotification(ctx context.Context, notif *Request) error {
	notif.ID = nil
	notif.JSONRPC = "2.0"

	select {
	case <-t.done:
		return ErrTransportClosed
	default:
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("mcp: sse marshal notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.postURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mcp: sse notification request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("mcp: sse notification: %w", err)
	}
	httpResp.Body.Close()

	return nil
}

// Close closes the SSE stream and cleans up pending requests.
func (t *SSETransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)

		if t.sseBody != nil {
			// Closing the body causes scanner.Scan() in readLoop to return false,
			// which triggers readLoop's deferred closePending().
			t.sseBody.Close()
		}
	})
	return nil
}

// readLoop continuously reads the SSE stream and routes responses to pending callers.
// It reuses the scanner created during connect.
func (t *SSETransport) readLoop() {
	defer t.closePending()

	var dataLines []string

	for t.scanner.Scan() {
		select {
		case <-t.done:
			return
		default:
		}

		line := t.scanner.Text()

		if line == "" {
			if len(dataLines) > 0 {
				data := strings.TrimSpace(strings.Join(dataLines, "\n"))
				t.handleEvent(data)
			}
			dataLines = nil
			continue
		}

		if val, ok := strings.CutPrefix(line, "data:"); ok {
			dataLines = append(dataLines, val)
		}
	}
}

func (t *SSETransport) handleEvent(data string) {
	if data == "" {
		return
	}

	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return
	}

	// Notification: no ID.
	if resp.ID == nil {
		var notif Notification
		if err := json.Unmarshal([]byte(data), &notif); err == nil && notif.Method != "" {
			if t.notifHandler != nil {
				t.notifHandler(notif.Method, notif.Params)
			}
		}
		return
	}

	t.mu.Lock()
	ch, ok := t.pending[*resp.ID]
	if ok {
		delete(t.pending, *resp.ID)
	}
	t.mu.Unlock()

	if ok {
		ch <- &resp
	}
}

// closePending closes all pending response channels. Safe to call multiple
// times from both Close() and readLoop — uses sync.Once internally.
func (t *SSETransport) closePending() {
	t.mu.Lock()
	for id, ch := range t.pending {
		close(ch)
		delete(t.pending, id)
	}
	t.mu.Unlock()
}
