package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FileReadTool reads files with optional offset and limit.
type FileReadTool struct{}

type fileReadInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // line number to start from (0-based)
	Limit  int    `json:"limit,omitempty"`  // max lines to read
}

func (t *FileReadTool) Name() string { return "file_read" }

func (t *FileReadTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var in fileReadInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if in.Path == "" {
		return &Result{Content: "path is required", IsError: true}, nil
	}

	f, err := os.Open(in.Path)
	if err != nil {
		return &Result{Content: fmt.Sprintf("cannot open file: %s", err), IsError: true}, nil
	}
	defer f.Close()

	if in.Limit == 0 {
		in.Limit = 2000
	}

	scanner := bufio.NewScanner(f)
	var lines []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum <= in.Offset {
			continue
		}
		if len(lines) >= in.Limit {
			break
		}
		lines = append(lines, fmt.Sprintf("%d\t%s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return &Result{Content: fmt.Sprintf("read error: %s", err), IsError: true}, nil
	}

	if len(lines) == 0 {
		return &Result{Content: "(empty file or offset beyond end)"}, nil
	}

	return &Result{Content: strings.Join(lines, "\n")}, nil
}
