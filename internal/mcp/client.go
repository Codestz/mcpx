package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client is a high-level MCP protocol client.
type Client struct {
	transport Transport
}

// NewClient creates a new MCP client using the given transport.
func NewClient(t Transport) *Client {
	return &Client{transport: t}
}

// Initialize performs the MCP handshake: sends an "initialize" request,
// validates the server response, then sends "notifications/initialized".
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcpx",
			"version": "0.1.0",
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

	// Send initialized notification.
	return c.transport.SendNotification(ctx, &Request{
		Method: "notifications/initialized",
	})
}

// listToolsResult is used to unmarshal the tools/list response.
type listToolsResult struct {
	Tools []Tool `json:"tools"`
}

// ListTools fetches the list of tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.transport.Send(ctx, &Request{
		Method: "tools/list",
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

	return result.Tools, nil
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

// Close closes the underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
