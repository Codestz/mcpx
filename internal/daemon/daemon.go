package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codestz/mcpx/internal/mcp"
)

// Daemon holds a single MCP server alive and accepts client connections
// over a unix socket. It proxies JSON-RPC requests to the server and
// routes responses back to the correct client.
type Daemon struct {
	serverName string
	socketPath string
	pidPath    string

	transport  *mcp.StdioTransport
	listener   net.Listener
	initResult json.RawMessage // cached InitializeResult from handshake

	idleTimeout  time.Duration
	lastActivity atomic.Value // stores time.Time

	done chan struct{}
	wg   sync.WaitGroup
}

// Start launches the daemon: starts the MCP server, performs the MCP handshake,
// listens on a unix socket, and serves client requests until shutdown.
// This function blocks until the daemon exits.
func Start(serverName string, scope string, command string, args []string, env []string, idleTimeout time.Duration) error {
	socketPath := SocketPath(serverName, scope)
	pidPath := PIDPath(serverName, scope)

	// Clean up stale socket.
	os.Remove(socketPath)

	// Start the real MCP server.
	transport, err := mcp.NewStdioTransport(command, args, env)
	if err != nil {
		return fmt.Errorf("daemon: start server %q: %w", serverName, err)
	}

	// MCP handshake — done once at daemon start.
	// We perform the handshake manually to capture the raw InitializeResult
	// so we can replay it to clients that send "initialize" requests.
	initResult, err := performHandshake(transport)
	if err != nil {
		transport.Close()
		return fmt.Errorf("daemon: initialize %q: %w", serverName, err)
	}

	// Listen on unix socket.
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		transport.Close()
		return fmt.Errorf("daemon: listen %s: %w", socketPath, err)
	}

	// Restrict socket permissions.
	os.Chmod(socketPath, 0600)

	// Write PID file.
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0600); err != nil {
		listener.Close()
		transport.Close()
		return fmt.Errorf("daemon: write pid: %w", err)
	}

	d := &Daemon{
		serverName:  serverName,
		socketPath:  socketPath,
		pidPath:     pidPath,
		transport:   transport,
		listener:    listener,
		initResult:  initResult,
		idleTimeout: idleTimeout,
		done:        make(chan struct{}),
	}
	d.lastActivity.Store(time.Now())

	log.Printf("daemon: %s listening on %s (pid %d, idle timeout %s)", serverName, socketPath, os.Getpid(), idleTimeout)

	// Signal handling.
	sigCh := make(chan os.Signal, 1)
	sigs := append([]os.Signal{os.Interrupt}, extraSignals()...)
	signal.Notify(sigCh, sigs...)
	go func() {
		select {
		case sig := <-sigCh:
			log.Printf("daemon: %s received %s, shutting down", serverName, sig)
			d.shutdown()
		case <-d.done:
		}
	}()

	// Transport death watcher: if the MCP server subprocess dies,
	// shut down the daemon instead of becoming a zombie.
	go func() {
		select {
		case <-d.transport.Dead():
			log.Printf("daemon: %s transport died, shutting down", serverName)
			d.shutdown()
		case <-d.done:
		}
	}()

	// Idle watcher.
	if idleTimeout > 0 {
		go d.idleWatcher()
	}

	// Accept loop.
	d.acceptLoop()

	// Wait for all connections to drain.
	d.wg.Wait()
	transport.Close()

	log.Printf("daemon: %s exited", serverName)
	return nil
}

func (d *Daemon) acceptLoop() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.done:
				return
			default:
				log.Printf("daemon: accept error: %v", err)
				continue
			}
		}

		d.lastActivity.Store(time.Now())
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.handleConn(conn)
		}()
	}
}

// handleConn reads JSON-RPC requests from a client, forwards them to the
// MCP server (via StdioTransport.Send), and writes responses back.
func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		d.lastActivity.Store(time.Now())

		var req mcp.Request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("daemon: malformed request: %v", err)
			continue
		}

		// Handle notifications (no ID, no response expected).
		if req.ID == nil {
			// Swallow notifications/initialized — the daemon already sent it.
			if req.Method == "notifications/initialized" {
				continue
			}
			_ = d.transport.SendNotification(context.Background(), &req)
			continue
		}

		// Save the client's original ID.
		clientID := *req.ID

		// Intercept "initialize" — return cached result instead of re-initializing.
		if req.Method == "initialize" {
			d.writeResponse(conn, &mcp.Response{
				JSONRPC: "2.0",
				ID:      &clientID,
				Result:  d.initResult,
			})
			continue
		}

		// Forward to the real MCP server. StdioTransport.Send assigns its own ID.
		resp, err := d.transport.Send(context.Background(), &req)
		if err != nil {
			// If the transport is dead, stop serving this connection.
			if errors.Is(err, mcp.ErrTransportClosed) {
				errResp := &mcp.Response{
					JSONRPC: "2.0",
					ID:      &clientID,
					Error: &mcp.RPCError{
						Code:    -32603,
						Message: fmt.Sprintf("daemon: server process died: %v", err),
					},
				}
				d.writeResponse(conn, errResp)
				return // stop serving, connection will close via defer
			}
			// Other errors: send error response but keep trying.
			errResp := &mcp.Response{
				JSONRPC: "2.0",
				ID:      &clientID,
				Error: &mcp.RPCError{
					Code:    -32603,
					Message: fmt.Sprintf("daemon: forward error: %v", err),
				},
			}
			d.writeResponse(conn, errResp)
			continue
		}

		// Remap to client's original ID.
		resp.ID = &clientID
		d.writeResponse(conn, resp)
	}
}

func (d *Daemon) writeResponse(conn net.Conn, resp *mcp.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("daemon: marshal response: %v", err)
		return
	}
	data = append(data, '\n')
	conn.Write(data)
}

func (d *Daemon) idleWatcher() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			last := d.lastActivity.Load().(time.Time)
			if time.Since(last) > d.idleTimeout {
				log.Printf("daemon: %s idle for %s, shutting down", d.serverName, d.idleTimeout)
				d.shutdown()
				return
			}
		case <-d.done:
			return
		}
	}
}

func (d *Daemon) shutdown() {
	select {
	case <-d.done:
		return // already shutting down
	default:
		close(d.done)
	}
	d.listener.Close()
	os.Remove(d.socketPath)
	os.Remove(d.pidPath)
}

// performHandshake sends the MCP initialize request and notifications/initialized,
// returning the raw InitializeResult JSON for replay to future clients.
func performHandshake(transport *mcp.StdioTransport) (json.RawMessage, error) {
	params := map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcpx",
			"version": "1.3.0",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := transport.Send(ctx, &mcp.Request{
		Method: "initialize",
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mcp.ErrInitFailed, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%w: %s", mcp.ErrInitFailed, resp.Error.Error())
	}

	// Cache the raw result.
	initResult := resp.Result

	// Send initialized notification.
	if err := transport.SendNotification(ctx, &mcp.Request{
		Method: "notifications/initialized",
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", mcp.ErrInitFailed, err)
	}

	return initResult, nil
}

// SocketPath returns the unix socket path for a server's daemon.
// Scope isolates daemons per project (e.g. hash of project root).
func SocketPath(serverName, scope string) string {
	if scope != "" {
		return fmt.Sprintf("/tmp/mcpx-%s-%s-%d.sock", serverName, scope, os.Getuid())
	}
	return fmt.Sprintf("/tmp/mcpx-%s-%d.sock", serverName, os.Getuid())
}

// PIDPath returns the PID file path for a server's daemon.
func PIDPath(serverName, scope string) string {
	if scope != "" {
		return fmt.Sprintf("/tmp/mcpx-%s-%s-%d.pid", serverName, scope, os.Getuid())
	}
	return fmt.Sprintf("/tmp/mcpx-%s-%d.pid", serverName, os.Getuid())
}

// LogPath returns the log file path for a server's daemon.
func LogPath(serverName, scope string) string {
	if scope != "" {
		return fmt.Sprintf("/tmp/mcpx-%s-%s-%d.log", serverName, scope, os.Getuid())
	}
	return fmt.Sprintf("/tmp/mcpx-%s-%d.log", serverName, os.Getuid())
}
