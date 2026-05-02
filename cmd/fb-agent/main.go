// Package main is the in-VM agent binary that executes tools inside Firecracker microVMs.
//
// fb-agent communicates with the host via vsock. It receives tool execution
// requests, runs them in the sandboxed VM environment, and reports results
// back to the host orchestrator.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/forgebox/forgebox/internal/agent"
	atools "github.com/forgebox/forgebox/internal/agent/tools"
)

const (
	// vsock port the agent listens on for host communication.
	vsockPort = 10000
)

func main() {
	slog.Info("fb-agent starting", "pid", os.Getpid())

	// Create the listener before the context so os.Exit doesn't skip defer cancel().
	listener, err := listenVsock(vsockPort)
	if err != nil {
		slog.Error("failed to listen on vsock", "port", vsockPort, "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	defer func() { _ = listener.Close() }()

	registry := atools.NewRegistry()
	registry.Register(&atools.BashTool{})
	registry.Register(&atools.FileReadTool{})
	registry.Register(&atools.FileWriteTool{})
	registry.Register(&atools.FileEditTool{})
	registry.Register(&atools.GlobTool{})
	registry.Register(&atools.GrepTool{})
	registry.Register(&atools.WebFetchTool{})

	// Management tools — call back to the gateway via FORGEBOX_API_URL/TOKEN
	// to manage ForgeBox itself (spec 5.0.0).
	registry.Register(&atools.ListAgentsTool{})
	registry.Register(&atools.GetAgentTool{})
	registry.Register(&atools.CreateAgentTool{})
	registry.Register(&atools.UpdateAgentTool{})
	registry.Register(&atools.DeleteAgentTool{})
	registry.Register(&atools.ListProvidersTool{})
	registry.Register(&atools.ListModelsForProviderTool{})

	a := agent.New(agent.Config{
		Tools:   registry,
		Workdir: "/workspace",
	})

	slog.Info("fb-agent ready", "vsock_port", vsockPort)

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("accept error", "error", err)
			continue
		}
		go a.HandleConnection(ctx, conn)
	}
}

// listenVsock creates a vsock listener. Falls back to TCP on localhost for
// development outside of Firecracker VMs.
func listenVsock(port int) (net.Listener, error) {
	// Try vsock first (inside a Firecracker VM).
	if _, err := os.Stat("/dev/vsock"); err == nil {
		return net.Listen("vsock", fmt.Sprintf(":%d", port))
	}

	// Fallback to TCP for development.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	slog.Warn("vsock not available, falling back to TCP", "addr", addr)
	return net.Listen("tcp", addr)
}
