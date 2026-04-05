# ForgeBox Plugin Development Guide

This guide walks you through creating plugins for ForgeBox. Plugins extend the
platform with new LLM providers, input channels, tools, and storage backends.

## Table of Contents

- [Plugin Types Overview](#plugin-types-overview)
- [Creating a Provider Plugin](#creating-a-provider-plugin)
- [Creating a Tool Plugin](#creating-a-tool-plugin)
- [Creating a Channel Plugin](#creating-a-channel-plugin)
- [Plugin Packaging](#plugin-packaging)
- [Testing Plugins](#testing-plugins)
- [Publishing Plugins](#publishing-plugins)

## Plugin Types Overview

ForgeBox defines four plugin interfaces in `pkg/sdk/`:

| Type       | Interface         | What It Does                                       |
|------------|-------------------|----------------------------------------------------|
| Provider   | `ProviderPlugin`  | Connects to an LLM API (completions, streaming)    |
| Tool       | `ToolPlugin`      | Defines a tool the LLM can invoke inside a microVM |
| Channel    | `ChannelPlugin`   | Ingests user requests from external systems         |
| Storage    | `StoragePlugin`   | Persists files and task artifacts                   |

All plugins share a common base:

```go
type Plugin interface {
    Name() string
    Init(config map[string]any) error
}
```

## Creating a Provider Plugin

Provider plugins integrate LLM APIs. Here is a complete example for a hypothetical
"AcmeLLM" provider.

```go
package acme

import (
    "context"
    "fmt"
    "io"
    "net/http"

    "github.com/forgebox-dev/forgebox/pkg/sdk"
)

// AcmeProvider implements sdk.ProviderPlugin.
type AcmeProvider struct {
    apiKey  string
    baseURL string
    client  *http.Client
}

func (p *AcmeProvider) Name() string {
    return "acme"
}

func (p *AcmeProvider) Init(config map[string]any) error {
    key, ok := config["api_key"].(string)
    if !ok || key == "" {
        return fmt.Errorf("acme: api_key is required")
    }
    p.apiKey = key
    p.baseURL = "https://api.acme-llm.com/v1"
    if base, ok := config["base_url"].(string); ok {
        p.baseURL = base
    }
    p.client = &http.Client{}
    return nil
}

func (p *AcmeProvider) Complete(ctx context.Context, req sdk.CompletionRequest) (sdk.CompletionResponse, error) {
    // Build and send the HTTP request to Acme's API.
    // Map sdk.CompletionRequest fields to Acme's request format.
    // Parse Acme's response into sdk.CompletionResponse.
    return sdk.CompletionResponse{}, fmt.Errorf("not implemented")
}

func (p *AcmeProvider) Stream(ctx context.Context, req sdk.CompletionRequest) (sdk.Stream, error) {
    // Similar to Complete, but return an sdk.Stream that yields tokens
    // incrementally. The Stream interface has Next() (sdk.StreamEvent, error).
    return nil, fmt.Errorf("not implemented")
}

func (p *AcmeProvider) ListModels(ctx context.Context) ([]sdk.Model, error) {
    return []sdk.Model{
        {ID: "acme-fast", Name: "Acme Fast", ContextWindow: 8192},
        {ID: "acme-large", Name: "Acme Large", ContextWindow: 128000},
    }, nil
}

func (p *AcmeProvider) Capabilities() sdk.ProviderCapabilities {
    return sdk.ProviderCapabilities{
        Streaming:      true,
        ToolCalling:    true,
        Vision:         false,
        MaxConcurrency: 10,
    }
}
```

Register your provider in `plugins/registry.go`:

```go
func init() {
    sdk.RegisterProvider("acme", func() sdk.ProviderPlugin {
        return &acme.AcmeProvider{}
    })
}
```

## Creating a Tool Plugin

Tool plugins define operations the LLM can invoke. Tools run inside the microVM.
The `Definition` method returns the tool's schema (name, description, parameters)
and `Execute` performs the operation.

```go
package wordcount

import (
    "context"
    "fmt"
    "strings"

    "github.com/forgebox-dev/forgebox/pkg/sdk"
)

// WordCountTool implements sdk.ToolPlugin.
type WordCountTool struct{}

func (t *WordCountTool) Name() string {
    return "word_count"
}

func (t *WordCountTool) Init(config map[string]any) error {
    return nil
}

func (t *WordCountTool) Definition() sdk.ToolDefinition {
    return sdk.ToolDefinition{
        Name:        "word_count",
        Description: "Count the number of words in a text string.",
        Parameters: sdk.Schema{
            Type: "object",
            Properties: map[string]sdk.Schema{
                "text": {
                    Type:        "string",
                    Description: "The text to count words in.",
                },
            },
            Required: []string{"text"},
        },
    }
}

func (t *WordCountTool) Execute(ctx context.Context, params map[string]any) (sdk.ToolResult, error) {
    text, ok := params["text"].(string)
    if !ok {
        return sdk.ToolResult{}, fmt.Errorf("word_count: 'text' parameter must be a string")
    }
    count := len(strings.Fields(text))
    return sdk.ToolResult{
        Output: fmt.Sprintf("Word count: %d", count),
    }, nil
}

func (t *WordCountTool) Validate(params map[string]any) error {
    if _, ok := params["text"].(string); !ok {
        return fmt.Errorf("word_count: 'text' parameter is required and must be a string")
    }
    return nil
}
```

Register your tool:

```go
func init() {
    sdk.RegisterTool("word_count", func() sdk.ToolPlugin {
        return &wordcount.WordCountTool{}
    })
}
```

### Tool Security Notes

- Tools execute inside the microVM. They have access only to `/workspace` and `/tmp`.
- Tools should never assume network access. If your tool needs the network, document
  it clearly and the task definition must grant `network:internet` plus domain
  allowlisting.
- Tools receive pre-validated parameters. However, always validate defensively inside
  `Execute` since the LLM may produce unexpected types.

## Creating a Channel Plugin

Channel plugins connect external systems (Slack, email, webhooks) to ForgeBox.
The `Listen` method starts receiving messages and sends them to the scheduler.
The `Send` method delivers responses back to the user.

```go
package webhookchannel

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/forgebox-dev/forgebox/pkg/sdk"
)

// WebhookChannel implements sdk.ChannelPlugin.
type WebhookChannel struct {
    listenAddr string
    handler    sdk.MessageHandler
}

func (c *WebhookChannel) Name() string {
    return "webhook"
}

func (c *WebhookChannel) Init(config map[string]any) error {
    addr, ok := config["listen_addr"].(string)
    if !ok {
        addr = ":9090"
    }
    c.listenAddr = addr
    return nil
}

func (c *WebhookChannel) Listen(ctx context.Context, handler sdk.MessageHandler) error {
    c.handler = handler
    mux := http.NewServeMux()
    mux.HandleFunc("POST /webhook", c.handleIncoming)

    server := &http.Server{Addr: c.listenAddr, Handler: mux}
    go func() {
        <-ctx.Done()
        server.Close()
    }()
    return server.ListenAndServe()
}

func (c *WebhookChannel) handleIncoming(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        UserID  string `json:"user_id"`
        Message string `json:"message"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    msg := sdk.IncomingMessage{
        ChannelName: "webhook",
        UserID:      payload.UserID,
        Content:     payload.Message,
    }
    if err := c.handler(r.Context(), msg); err != nil {
        http.Error(w, "processing failed", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusAccepted)
}

func (c *WebhookChannel) Send(ctx context.Context, msg sdk.OutgoingMessage) error {
    // Webhooks are fire-and-forget inbound. For response delivery,
    // you could POST to a callback URL provided in the original message.
    return fmt.Errorf("webhook channel does not support outbound messages")
}

func (c *WebhookChannel) Capabilities() sdk.ChannelCapabilities {
    return sdk.ChannelCapabilities{
        SupportsThreads:    false,
        SupportsAttachments: false,
        SupportsReactions:   false,
    }
}
```

## Plugin Packaging

ForgeBox supports two plugin packaging models.

### Go Plugin (Shared Object)

Build your plugin as a Go shared object. This is the simplest approach for plugins
written in Go.

```bash
go build -buildmode=plugin -o myplugin.so ./plugins/myplugin/
```

Place the `.so` file in the ForgeBox plugins directory (default: `/etc/forgebox/plugins/`).
ForgeBox loads it at startup.

**Requirements:**
- Must be compiled with the same Go version as ForgeBox.
- Must be compiled on the same OS and architecture.
- Must export a `New` function that returns the appropriate plugin interface.

### gRPC Sidecar

For plugins written in other languages or when you need process isolation, package
your plugin as a standalone binary that implements the ForgeBox plugin gRPC service.

The protobuf definitions are in `pkg/sdk/proto/`. Generate client/server stubs for
your language:

```bash
protoc --go_out=. --go-grpc_out=. pkg/sdk/proto/plugin.proto
```

Configure ForgeBox to connect to your sidecar in `forgebox.yaml`:

```yaml
plugins:
  - name: my-python-tool
    type: tool
    transport: grpc
    address: localhost:50051
```

The gRPC sidecar approach is recommended for production deployments because it
provides process isolation and allows independent version management.

## Testing Plugins

### Unit Tests

Write standard Go unit tests. Mock the `sdk` types as needed.

```go
func TestWordCount_Execute(t *testing.T) {
    tool := &WordCountTool{}
    require.NoError(t, tool.Init(nil))

    tests := []struct {
        name     string
        params   map[string]any
        expected string
        wantErr  bool
    }{
        {
            name:     "simple sentence",
            params:   map[string]any{"text": "hello world"},
            expected: "Word count: 2",
        },
        {
            name:     "empty string",
            params:   map[string]any{"text": ""},
            expected: "Word count: 0",
        },
        {
            name:    "missing text parameter",
            params:  map[string]any{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := tool.Execute(context.Background(), tt.params)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.Output)
        })
    }
}
```

### Integration Tests

For plugins that interact with external services (provider plugins calling LLM APIs,
channel plugins listening on ports), write integration tests behind the `integration`
build tag. Use environment variables for credentials:

```go
//go:build integration

func TestAcmeProvider_Complete(t *testing.T) {
    apiKey := os.Getenv("ACME_API_KEY")
    if apiKey == "" {
        t.Skip("ACME_API_KEY not set")
    }
    p := &AcmeProvider{}
    require.NoError(t, p.Init(map[string]any{"api_key": apiKey}))
    // ...
}
```

### Plugin Test Harness

ForgeBox provides a test harness in `pkg/sdk/testing/` that validates your plugin
against the interface contract:

```go
func TestAcmeProvider_Contract(t *testing.T) {
    sdktesting.ValidateProvider(t, &acme.AcmeProvider{}, map[string]any{
        "api_key": "test-key",
    })
}
```

The harness checks that `Init` succeeds, `Name` returns a non-empty string,
`Capabilities` returns valid values, and the plugin handles cancellation correctly.

## Publishing Plugins

### Community Plugin Registry

We are building a community plugin registry. Until it launches, share your plugins as
Git repositories with the naming convention `forgebox-plugin-<name>`.

### Plugin Repository Structure

```
forgebox-plugin-acme/
  plugin.go          # Plugin implementation
  plugin_test.go     # Tests
  go.mod             # Must depend on github.com/forgebox-dev/forgebox/pkg/sdk
  README.md          # Usage instructions, configuration options
  LICENSE            # Must be compatible with Apache 2.0
```

### Compatibility

Pin your `pkg/sdk` dependency to a specific ForgeBox release. The plugin SDK follows
semantic versioning. Minor version bumps are backward compatible. Major version bumps
may break plugin compatibility and will include a migration guide.

```bash
go get github.com/forgebox-dev/forgebox/pkg/sdk@v0.5.0
```
