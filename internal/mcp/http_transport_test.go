package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHTTPTransport_JSONResponse(t *testing.T) {
	var reqCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		reqCount.Add(1)

		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
		}
		result, _ := json.Marshal(map[string]any{"status": "ok"})
		resp.Result = result

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "test-session-123")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.URL, nil)
	defer transport.Close()

	ctx := context.Background()
	resp, err := transport.Send(ctx, &Request{Method: "test"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify session ID was captured.
	if transport.sessionID != "test-session-123" {
		t.Errorf("session ID = %q, want %q", transport.sessionID, "test-session-123")
	}
}

func TestHTTPTransport_SSEResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		result, _ := json.Marshal(map[string]any{"tools": []any{}})
		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
		respData, _ := json.Marshal(resp)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter is not a Flusher")
		}

		// Send a notification first (should be handled, not returned).
		notif, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/progress",
			"params":  map[string]any{"progress": 50},
		})
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", notif)
		flusher.Flush()

		// Send the actual response.
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", respData)
		flusher.Flush()
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.URL, nil)
	defer transport.Close()

	var gotNotification bool
	transport.SetNotificationHandler(func(method string, params json.RawMessage) {
		gotNotification = true
	})

	ctx := context.Background()
	resp, err := transport.Send(ctx, &Request{Method: "tools/list"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if !gotNotification {
		t.Error("notification handler was not called")
	}
}

func TestHTTPTransport_SessionReInit(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		count := callCount.Add(1)

		// First call with session: return 404 to trigger re-init.
		if count == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{JSONRPC: "2.0", ID: req.ID}
		result, _ := json.Marshal(map[string]any{"ok": true})
		resp.Result = result

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.URL, nil)
	transport.sessionID = "expired-session"
	defer transport.Close()

	ctx := context.Background()
	resp, err := transport.Send(ctx, &Request{Method: "test"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls (404 + retry), got %d", callCount.Load())
	}
}

func TestHTTPTransport_SendNotification(t *testing.T) {
	var gotNotif bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		gotNotif = true
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.URL, nil)
	defer transport.Close()

	ctx := context.Background()
	err := transport.SendNotification(ctx, &Request{Method: "notifications/initialized"})
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	if !gotNotif {
		t.Error("server did not receive notification")
	}
}

func TestHTTPTransport_CustomHeaders(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		gotAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{JSONRPC: "2.0", ID: req.ID}
		result, _ := json.Marshal(map[string]any{"ok": true})
		resp.Result = result

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transport := NewHTTPTransport(server.URL, map[string]string{
		"Authorization": "Bearer test-token",
	})
	defer transport.Close()

	ctx := context.Background()
	_, err := transport.Send(ctx, &Request{Method: "test"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
}
