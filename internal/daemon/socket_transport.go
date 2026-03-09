package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/codestz/mcpx/internal/mcp"
)

// SocketTransport implements mcp.Transport over a unix socket connection
// to a running mcpx daemon. Closing it only closes the socket — the daemon stays alive.
type SocketTransport struct {
	conn   net.Conn
	reader *bufio.Scanner

	pending map[int64]chan *mcp.Response
	mu      sync.Mutex
	nextID  atomic.Int64

	writeMu   sync.Mutex
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// NewSocketTransport connects to a daemon's unix socket and returns a transport.
func NewSocketTransport(socketPath string) (*SocketTransport, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("daemon: connect %s: %w", socketPath, err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	t := &SocketTransport{
		conn:    conn,
		reader:  scanner,
		pending: make(map[int64]chan *mcp.Response),
		done:    make(chan struct{}),
	}

	go t.readLoop()

	return t, nil
}

// Send sends a request over the socket and waits for the response.
func (t *SocketTransport) Send(ctx context.Context, req *mcp.Request) (*mcp.Response, error) {
	id := t.nextID.Add(1)
	req.ID = &id
	req.JSONRPC = "2.0"

	ch := make(chan *mcp.Response, 1)

	t.mu.Lock()
	select {
	case <-t.done:
		t.mu.Unlock()
		return nil, mcp.ErrTransportClosed
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
			return nil, mcp.ErrTransportClosed
		}
		return resp, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	case <-t.done:
		return nil, mcp.ErrTransportClosed
	}
}

// SendNotification sends a notification (no response expected).
func (t *SocketTransport) SendNotification(ctx context.Context, notif *mcp.Request) error {
	notif.ID = nil
	notif.JSONRPC = "2.0"

	select {
	case <-t.done:
		return mcp.ErrTransportClosed
	default:
	}

	return t.writeJSON(notif)
}

// Close closes the socket connection. The daemon stays alive.
func (t *SocketTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
		t.closeErr = t.conn.Close()

		t.mu.Lock()
		for id, ch := range t.pending {
			close(ch)
			delete(t.pending, id)
		}
		t.mu.Unlock()
	})
	return t.closeErr
}

func (t *SocketTransport) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("daemon: marshal: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	data = append(data, '\n')
	if _, err := t.conn.Write(data); err != nil {
		return fmt.Errorf("daemon: write: %w", err)
	}
	return nil
}

func (t *SocketTransport) readLoop() {
	defer func() {
		t.mu.Lock()
		for id, ch := range t.pending {
			close(ch)
			delete(t.pending, id)
		}
		t.mu.Unlock()
	}()

	for t.reader.Scan() {
		line := t.reader.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp mcp.Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		if resp.ID == nil {
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
