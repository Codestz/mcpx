package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// JSON-RPC 2.0 types

// Request represents a JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *int64           `json:"id"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error (code %d): %s", e.Code, e.Message)
}

// MCP protocol types

// Tool describes an MCP tool exposed by a server.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema describes the JSON Schema for a tool's input.
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySchema describes a single property in a tool's input schema.
type PropertySchema struct {
	Type        string          `json:"type"`
	Description string          `json:"description"`
	Default     any             `json:"default,omitempty"`
	Enum        []any           `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

// CallResult is the result of a tools/call request.
type CallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
}

// Content represents a single content block in a call result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Sentinel errors.
var (
	ErrToolNotFound    = errors.New("mcp: tool not found")
	ErrInitFailed      = errors.New("mcp: initialization failed")
	ErrTransportClosed = errors.New("mcp: transport closed")
)

// ToolError represents an error reported by an MCP tool (isError=true).
type ToolError struct {
	Name    string
	Message string
	Code    int
}

func (e *ToolError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("tool %q error (code %d): %s", e.Name, e.Code, e.Message)
	}
	return fmt.Sprintf("tool %q error: %s", e.Name, e.Message)
}
