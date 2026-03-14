package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client is a high-level MCP protocol client.
type Client struct {
	transport  Transport
	serverCaps ServerCapabilities
	serverInfo ServerInfo
}

// NewClient creates a new MCP client using the given transport.
func NewClient(t Transport) *Client {
	return &Client{transport: t}
}

// Initialize performs the MCP handshake: sends an "initialize" request,
// validates the server response, then sends "notifications/initialized".
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcpx",
			"version": "1.1.0",
		},
	}

	resp, err := c.transport.Send(ctx, &Request{
		Method: "initialize",
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInitFailed, err)
	}

	if resp.Error != nil {
		return fmt.Errorf("%w: %s", ErrInitFailed, resp.Error.Error())
	}

	// Parse server capabilities and info.
	var result InitializeResult
	if resp.Result != nil {
		if err := json.Unmarshal(resp.Result, &result); err == nil {
			c.serverCaps = result.Capabilities
			c.serverInfo = result.ServerInfo
		}
	}

	// Send initialized notification.
	return c.transport.SendNotification(ctx, &Request{
		Method: "notifications/initialized",
	})
}

// ServerCapabilities returns the capabilities reported by the server.
func (c *Client) ServerCapabilities() ServerCapabilities {
	return c.serverCaps
}

// ServerInfo returns the server identity information.
func (c *Client) ServerInfo() ServerInfo {
	return c.serverInfo
}

// listToolsResult is used to unmarshal the tools/list response.
type listToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// ListTools fetches the list of tools from the MCP server, handling pagination.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var allTools []Tool
	var cursor *string

	for {
		params := map[string]any{}
		if cursor != nil {
			params["cursor"] = *cursor
		}

		var reqParams any
		if len(params) > 0 {
			reqParams = params
		}

		resp, err := c.transport.Send(ctx, &Request{
			Method: "tools/list",
			Params: reqParams,
		})
		if err != nil {
			return nil, fmt.Errorf("mcp: list tools: %w", err)
		}

		if resp.Error != nil {
			return nil, resp.Error
		}

		var result listToolsResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("mcp: unmarshal tools: %w", err)
		}

		allTools = append(allTools, result.Tools...)

		if result.NextCursor == nil || *result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allTools, nil
}

// CallTool invokes a tool on the MCP server with the given arguments.
// If the server reports a tool-level error (isError=true), a *ToolError is returned.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*CallResult, error) {
	params := map[string]any{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.transport.Send(ctx, &Request{
		Method: "tools/call",
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("mcp: call tool %q: %w", name, err)
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	var result CallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: unmarshal call result: %w", err)
	}

	if result.IsError {
		msg := "unknown error"
		if len(result.Content) > 0 {
			msg = result.Content[0].Text
		}
		return &result, &ToolError{
			Name:    name,
			Message: msg,
		}
	}

	return &result, nil
}

// Ping sends a ping request to the server and returns an error if unreachable.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.transport.Send(ctx, &Request{
		Method: "ping",
	})
	if err != nil {
		return fmt.Errorf("mcp: ping: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Close closes the underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
