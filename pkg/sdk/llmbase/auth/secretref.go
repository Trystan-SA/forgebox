// Package auth holds vendor-agnostic credential strategies and secret-reference
// resolution for LLM provider plugins.
package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ResolveSecret resolves a credential reference to its concrete value.
//
// Accepted forms:
//   - "sk-..." (literal): returned unchanged.
//   - "env://NAME": value of os.Getenv("NAME"). Errors if unset.
//   - "file:///abs/path": contents of the file, whitespace-trimmed.
//   - "exec://command args...": stdout of the command, whitespace-trimmed.
//
// An empty input returns an error.
func ResolveSecret(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty secret reference")
	}
	switch {
	case strings.HasPrefix(ref, "env://"):
		name := strings.TrimPrefix(ref, "env://")
		v := os.Getenv(name)
		if v == "" {
			return "", fmt.Errorf("env var %q is empty or unset", name)
		}
		return v, nil
	case strings.HasPrefix(ref, "file://"):
		path := strings.TrimPrefix(ref, "file://")
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read secret file %q: %w", path, err)
		}
		return strings.TrimSpace(string(b)), nil
	case strings.HasPrefix(ref, "exec://"):
		cmdLine := strings.TrimPrefix(ref, "exec://")
		parts := strings.Fields(cmdLine)
		if len(parts) == 0 {
			return "", fmt.Errorf("empty exec secret command")
		}
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("exec secret %q: %w", cmdLine, err)
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return ref, nil
	}
}