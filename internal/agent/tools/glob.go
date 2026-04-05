package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// GlobTool finds files matching a glob pattern.
type GlobTool struct{}

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func (t *GlobTool) Name() string { return "glob" }

func (t *GlobTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var in globInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if in.Pattern == "" {
		return &Result{Content: "pattern is required", IsError: true}, nil
	}

	base := in.Path
	if base == "" {
		base = "."
	}

	fullPattern := filepath.Join(base, in.Pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return &Result{Content: fmt.Sprintf("glob error: %s", err), IsError: true}, nil
	}

	if len(matches) == 0 {
		return &Result{Content: "no matches found"}, nil
	}

	return &Result{Content: strings.Join(matches, "\n")}, nil
}
