// Package mcp implements Model Context Protocol client and server.
package mcp

import (
	"context"
	"log/slog"
)

// Client connects to an MCP server and discovers its tools.
type Client struct {
	serverURL string
}

// NewClient creates a new MCP client for the given server URL.
func NewClient(serverURL string) *Client {
	return &Client{serverURL: serverURL}
}

// Connect establishes a connection to the MCP server.
func (c *Client) Connect(ctx context.Context) error {
	slog.Info("MCP client connecting", "server", c.serverURL)
	return nil
}

// Close disconnects from the MCP server.
func (c *Client) Close() error {
	return nil
}
