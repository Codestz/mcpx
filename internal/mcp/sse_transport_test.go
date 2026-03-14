package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSSETransport_SendAndReceive(t *testing.T) {
	var mu sync.Mutex
	var postURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter is not a Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)

			mu.Lock()
			pURL := postURL
			mu.Unlock()

			// Send endpoint event.
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", pURL)
			flusher.Flush()

			// Keep stream open until client disconnects.
			<-r.Context().Done()
			return
		}

		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var req Request
			json.Unmarshal(body, &req)

			// This is a simplified test — in real SSE, the response comes on the GET stream.
			// For testing, we just accept the POST.
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	mu.Lock()
	postURL = server.URL + "/message"
	mu.Unlock()

	// For the SSE transport test, we need a server that sends responses on the SSE stream.
	// This requires a more sophisticated test setup. Let's test the basic connection flow.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will fail because the test server doesn't send responses on the SSE stream,
	// but it validates the connection and endpoint discovery flow.
	transport, err := NewSSETransport(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("NewSSETransport: %v", err)
	}
	defer transport.Close()

	if transport.postURL != server.URL+"/message" {
		t.Errorf("postURL = %q, want %q", transport.postURL, server.URL+"/message")
	}
}

func TestSSETransport_FullRoundtrip(t *testing.T) {
	// Channel to deliver responses on the SSE stream.
	responseCh := make(chan []byte, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter is not a Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send endpoint event with relative URL.
			fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
			flusher.Flush()

			// Stream responses from channel.
			for {
				select {
				case data := <-responseCh:
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
					flusher.Flush()
				case <-r.Context().Done():
					return
				}
			}
		}

		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var req Request
			json.Unmarshal(body, &req)

			// Build response and send on SSE stream.
			result, _ := json.Marshal(map[string]any{"tools": []any{}})
			resp := Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			}
			respData, _ := json.Marshal(resp)
			responseCh <- respData

			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport, err := NewSSETransport(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("NewSSETransport: %v", err)
	}
	defer transport.Close()

	// Verify relative URL resolution.
	if transport.postURL != server.URL+"/message" {
		t.Fatalf("postURL = %q, want %q", transport.postURL, server.URL+"/message")
	}

	resp, err := transport.Send(ctx, &Request{Method: "tools/list"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestSSETransport_Notification(t *testing.T) {
	responseCh := make(chan []byte, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("ResponseWriter is not a Flusher")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, "event: endpoint\ndata: /msg\n\n")
			flusher.Flush()

			for {
				select {
				case data := <-responseCh:
					fmt.Fprintf(w, "data: %s\n\n", data)
					flusher.Flush()
				case <-r.Context().Done():
					return
				}
			}
		}

		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var req Request
			json.Unmarshal(body, &req)

			// Send a server notification before the response.
			notif, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"method":  "notifications/tools/list_changed",
			})
			responseCh <- notif

			// Then send the actual response.
			result, _ := json.Marshal(map[string]any{"status": "ok"})
			resp := Response{JSONRPC: "2.0", ID: req.ID, Result: result}
			respData, _ := json.Marshal(resp)
			responseCh <- respData

			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport, err := NewSSETransport(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("NewSSETransport: %v", err)
	}
	defer transport.Close()

	var gotNotification bool
	transport.SetNotificationHandler(func(method string, params json.RawMessage) {
		if method == "notifications/tools/list_changed" {
			gotNotification = true
		}
	})

	resp, err := transport.Send(ctx, &Request{Method: "test"})
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
