package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// BashTool executes shell commands inside the VM.
type BashTool struct{}

type bashInput struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // seconds
}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var in bashInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if in.Command == "" {
		return &Result{Content: "command is required", IsError: true}, nil
	}

	timeout := 120 * time.Second
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "bash", "-c", in.Command) //nolint:gosec // intentional: runs inside a sandboxed VM
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	if err != nil {
		return &Result{
			Content: fmt.Sprintf("exit code %d\n%s", cmd.ProcessState.ExitCode(), output),
			IsError: true,
		}, nil
	}

	return &Result{Content: output}, nil
}
