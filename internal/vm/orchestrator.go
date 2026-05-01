// Package vm manages task execution isolation.
//
// In production (mode: "firecracker"), the orchestrator maintains a pool of
// pre-booted Firecracker microVMs. In development (mode: "local"), tools
// execute directly in the host process with no isolation.
package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/google/uuid"
)

// Orchestrator manages tool execution environments.
type Orchestrator struct {
	cfg    config.VMConfig
	mode   string // "local" or "firecracker"
	local  *LocalExecutor
	mu     sync.Mutex
	pool   []*VM
	active map[string]*VM
}

// VM represents a running Firecracker microVM.
type VM struct {
	ID        string
	Status    Status
	Config    AllocRequest
	BootedAt  time.Time
	AgentAddr string
	EnvVars   map[string]string
}

// Status tracks the lifecycle state of a VM.
type Status string

// VM status constants.
const (
	VMReady    Status = "ready"
	VMRunning  Status = "running"
	VMStopping Status = "stopping"
)

// AllocRequest configures a new VM allocation.
type AllocRequest struct {
	MemoryMB           int
	VCPUs              int
	Timeout            time.Duration
	NetworkAccess      bool
	ControlPlaneAccess bool              // grants narrow egress to FORGEBOX_API_URL only
	EnvVars            map[string]string // injected as guest env (e.g. FORGEBOX_API_TOKEN)
}

// ExecResult is the output of a tool execution.
type ExecResult struct {
	Content    string
	IsError    bool
	DurationMS int64
}

// NewOrchestrator creates an orchestrator based on the configured mode.
func NewOrchestrator(cfg config.VMConfig) (*Orchestrator, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = "local"
	}

	o := &Orchestrator{
		cfg:    cfg,
		mode:   mode,
		active: make(map[string]*VM),
	}

	if mode == "local" {
		workdir, _ := os.Getwd()
		o.local = NewLocalExecutor(workdir)
		slog.Info("VM orchestrator starting in LOCAL mode (no isolation)")
		return o, nil
	}

	// Firecracker mode — pre-boot the VM pool.
	slog.Info("VM orchestrator starting in FIRECRACKER mode", "pool_size", cfg.PoolSize)
	for i := 0; i < cfg.PoolSize; i++ {
		vm, err := o.bootVM(context.Background(), &AllocRequest{
			MemoryMB: cfg.DefaultMemoryMB,
			VCPUs:    cfg.DefaultVCPUs,
		})
		if err != nil {
			slog.Warn("failed to pre-boot VM", "index", i, "error", err)
			continue
		}
		o.pool = append(o.pool, vm)
	}
	slog.Info("VM pool ready", "available", len(o.pool))

	return o, nil
}

// Allocate assigns a VM from the pool or boots a fresh one.
// In local mode, returns a virtual VM ID immediately.
func (o *Orchestrator) Allocate(ctx context.Context, req *AllocRequest) (string, error) {
	if req.Timeout == 0 {
		req.Timeout = o.cfg.DefaultTimeout
	}

	if o.mode == "local" {
		id := "local-" + uuid.New().String()[:8]
		o.mu.Lock()
		o.active[id] = &VM{
			ID:       id,
			Status:   VMRunning,
			Config:   *req,
			BootedAt: time.Now(),
			EnvVars:  req.EnvVars,
		}
		o.mu.Unlock()
		return id, nil
	}

	// Firecracker mode.
	o.mu.Lock()
	defer o.mu.Unlock()

	if req.MemoryMB == 0 {
		req.MemoryMB = o.cfg.DefaultMemoryMB
	}
	if req.VCPUs == 0 {
		req.VCPUs = o.cfg.DefaultVCPUs
	}

	if len(o.pool) > 0 {
		vm := o.pool[len(o.pool)-1]
		o.pool = o.pool[:len(o.pool)-1]
		vm.Status = VMRunning
		vm.Config = *req
		vm.EnvVars = req.EnvVars
		o.active[vm.ID] = vm
		go o.replenishPool()
		slog.Debug("allocated VM from pool", "vm_id", vm.ID)
		return vm.ID, nil
	}

	slog.Warn("VM pool exhausted, booting on-demand")
	vm, err := o.bootVM(ctx, req)
	if err != nil {
		return "", fmt.Errorf("boot VM: %w", err)
	}
	vm.Status = VMRunning
	o.active[vm.ID] = vm
	return vm.ID, nil
}

// Execute runs a tool inside the specified VM (or locally in dev mode).
func (o *Orchestrator) Execute(ctx context.Context, vmID, toolName string, input json.RawMessage) (*ExecResult, error) {
	o.mu.Lock()
	vm, ok := o.active[vmID]
	o.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	execCtx, cancel := context.WithTimeout(ctx, vm.Config.Timeout)
	defer cancel()

	if o.mode == "local" {
		return o.local.Execute(execCtx, toolName, input, vm.EnvVars)
	}

	// Firecracker mode — call the in-VM agent via vsock.
	// Env vars are stored on *VM but are not yet wired into the firecracker
	// guest config; that is deferred to a follow-up commit.
	if len(vm.EnvVars) > 0 {
		slog.Debug("VM env injection deferred (firecracker mode)", "vm_id", vm.ID, "env_count", len(vm.EnvVars))
	}
	start := time.Now()
	result, err := o.callAgent(execCtx, vm, toolName, input)
	if err != nil {
		return nil, fmt.Errorf("agent call: %w", err)
	}
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

// Release destroys a VM and returns resources.
func (o *Orchestrator) Release(ctx context.Context, vmID string) {
	o.mu.Lock()
	vm, ok := o.active[vmID]
	if ok {
		delete(o.active, vmID)
	}
	o.mu.Unlock()

	if !ok {
		return
	}

	if o.mode == "local" {
		slog.Debug("released local executor", "vm_id", vmID)
		return
	}

	vm.Status = VMStopping
	if err := o.destroyVM(ctx, vm); err != nil {
		slog.Error("failed to destroy VM", "vm_id", vmID, "error", err)
	}
	slog.Debug("released VM", "vm_id", vmID)
}

// Shutdown stops all active VMs and drains the pool.
func (o *Orchestrator) Shutdown(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.mode == "local" {
		o.active = make(map[string]*VM)
		slog.Info("local orchestrator shut down")
		return
	}

	for id, vm := range o.active {
		if err := o.destroyVM(ctx, vm); err != nil {
			slog.Error("shutdown: failed to destroy VM", "vm_id", id, "error", err)
		}
	}
	for _, vm := range o.pool {
		if err := o.destroyVM(ctx, vm); err != nil {
			slog.Error("shutdown: failed to destroy pooled VM", "vm_id", vm.ID, "error", err)
		}
	}
	o.active = make(map[string]*VM)
	o.pool = nil
	slog.Info("VM orchestrator shut down")
}

// Status returns pool and active VM counts.
func (o *Orchestrator) Status() (poolSize, activeCount int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.pool), len(o.active)
}

// --- Firecracker internals (only used when mode == "firecracker") ---

func (o *Orchestrator) bootVM(ctx context.Context, req *AllocRequest) (*VM, error) {
	id := uuid.New().String()[:12]
	slog.Debug("booting VM", "vm_id", id, "memory_mb", req.MemoryMB, "vcpus", req.VCPUs)
	slog.Debug("VM control plane access", "vm_id", id, "enabled", req.ControlPlaneAccess)

	vm := &VM{
		ID:        id,
		Status:    VMReady,
		Config:    *req,
		BootedAt:  time.Now(),
		AgentAddr: fmt.Sprintf("vsock://%s:%d", id, 10000),
		EnvVars:   req.EnvVars,
	}

	// TODO: Firecracker SDK calls:
	// machine, _ := firecracker.NewMachine(ctx, firecrackerCfg)
	// machine.Start(ctx)
	// waitForAgent(ctx, vm)

	return vm, nil
}

func (o *Orchestrator) destroyVM(ctx context.Context, vm *VM) error {
	slog.Debug("destroying VM", "vm_id", vm.ID)
	// TODO: machine.Shutdown(ctx) + cleanup overlay + release network
	return nil
}

func (o *Orchestrator) callAgent(ctx context.Context, vm *VM, toolName string, input json.RawMessage) (*ExecResult, error) {
	// TODO: gRPC-over-vsock call to fb-agent
	return nil, fmt.Errorf("firecracker agent communication not yet implemented")
}

func (o *Orchestrator) replenishPool() {
	o.mu.Lock()
	needed := o.cfg.PoolSize - len(o.pool)
	o.mu.Unlock()

	for i := 0; i < needed; i++ {
		vm, err := o.bootVM(context.Background(), &AllocRequest{
			MemoryMB: o.cfg.DefaultMemoryMB,
			VCPUs:    o.cfg.DefaultVCPUs,
		})
		if err != nil {
			slog.Warn("failed to replenish pool", "error", err)
			return
		}
		o.mu.Lock()
		o.pool = append(o.pool, vm)
		o.mu.Unlock()
	}
}
