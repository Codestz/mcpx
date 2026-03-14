package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// NotificationHandler is a callback for server-initiated notifications.
type NotificationHandler func(method string, params json.RawMessage)

// Transport defines the interface for communicating with an MCP server.
type Transport interface {
	// Send sends a request and waits for the corresponding response.
	Send(ctx context.Context, req *Request) (*Response, error)
	// SendNotification sends a notification (no response expected).
	SendNotification(ctx context.Context, notif *Request) error
	// Close shuts down the transport and releases resources.
	Close() error
}

// NotificationAware is optionally implemented by transports that support
// server-initiated notification callbacks.
type NotificationAware interface {
	SetNotificationHandler(handler NotificationHandler)
}

// StdioTransport communicates with an MCP server subprocess via stdin/stdout JSON-RPC.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bytes.Buffer

	// Response routing: pending maps request IDs to response channels.
	pending map[int64]chan *Response
	mu      sync.Mutex
	nextID  atomic.Int64

	writeMu   sync.Mutex // protects writes to stdin
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error

	// dead is closed when the subprocess's stdout pipe closes (readLoop exits).
	// Separate from done to avoid double-close: done = "intentional close",
	// dead = "subprocess died unexpectedly".
	dead     chan struct{}
	deadOnce sync.Once

	notifHandler NotificationHandler
}

// NewStdioTransport spawns a subprocess and returns a transport that communicates
// with it over stdin/stdout using line-delimited JSON-RPC.
func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start %q: %w", command, err)
	}

	scanner := bufio.NewScanner(stdout)
	// MCP responses can be large; allow up to 1MB per line.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	t := &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  scanner,
		stderr:  &stderr,
		pending: make(map[int64]chan *Response),
		done:    make(chan struct{}),
		dead:    make(chan struct{}),
	}

	go t.readLoop()

	return t, nil
}

// Send assigns an ID to the request, sends it, and waits for the response.
func (t *StdioTransport) Send(ctx context.Context, req *Request) (*Response, error) {
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

	if err := t.writeJSON(req); err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, err
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

// SendNotification sends a JSON-RPC notification (no ID, no response expected).
func (t *StdioTransport) SendNotification(ctx context.Context, notif *Request) error {
	notif.ID = nil
	notif.JSONRPC = "2.0"

	select {
	case <-t.done:
		return ErrTransportClosed
	default:
	}

	return t.writeJSON(notif)
}

// Close shuts down the transport: closes stdin, waits for the process to exit
// (with a 5-second timeout), and kills it if necessary.
func (t *StdioTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		t.stdin.Close()

		exited := make(chan error, 1)
		go func() {
			exited <- t.cmd.Wait()
		}()

		select {
		case err := <-exited:
			t.closeErr = err
		case <-time.After(5 * time.Second):
			t.cmd.Process.Kill()
			<-exited
			t.closeErr = fmt.Errorf("mcp: process killed after timeout")
		}

		// Close all pending channels so blocked Send calls unblock.
		t.mu.Lock()
		for id, ch := range t.pending {
			close(ch)
			delete(t.pending, id)
		}
		t.mu.Unlock()
	})
	return t.closeErr
}

// writeJSON marshals v to JSON and writes it as a single line to stdin.
func (t *StdioTransport) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("mcp: marshal: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("mcp: write: %w", err)
	}
	return nil
}

// SetNotificationHandler sets a callback for server-initiated notifications.
func (t *StdioTransport) SetNotificationHandler(handler NotificationHandler) {
	t.notifHandler = handler
}

// Dead returns a channel that is closed when the subprocess's stdout pipe closes,
// indicating the MCP server process has died unexpectedly.
func (t *StdioTransport) Dead() <-chan struct{} {
	return t.dead
}

// readLoop reads JSON-RPC responses from stdout and routes them to pending callers.
func (t *StdioTransport) readLoop() {
	defer func() {
		// Signal that the subprocess died.
		t.deadOnce.Do(func() { close(t.dead) })

		// On exit, close all pending channels.
		t.mu.Lock()
		for id, ch := range t.pending {
			close(ch)
			delete(t.pending, id)
		}
		t.mu.Unlock()
	}()

	for t.stdout.Scan() {
		line := t.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			// Skip malformed lines.
			continue
		}

		if resp.ID == nil {
			// Server-initiated notification.
			var notif Notification
			if err := json.Unmarshal(line, &notif); err == nil && notif.Method != "" {
				if t.notifHandler != nil {
					t.notifHandler(notif.Method, notif.Params)
				}
			}
			continue
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
}
