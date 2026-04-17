package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/forgebox/forgebox/pkg/sdk"
	"gopkg.in/yaml.v3"
)

// automationDoc is the YAML-facing shape of an automation. It flattens the
// canvas state (nodes, edges) and hoists the trigger for readability. This is
// the format a human authors or reviews; the gateway is authoritative for how
// it maps to and from stored fields.
type automationDoc struct {
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description,omitempty"`
	Sharing     string                   `yaml:"sharing"`
	Enabled     bool                     `yaml:"enabled"`
	Trigger     map[string]any           `yaml:"trigger,omitempty"`
	Nodes       []map[string]any         `yaml:"nodes"`
	Edges       []map[string]any         `yaml:"edges"`
}

type flowNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Position map[string]any `json:"position"`
	Data     map[string]any `json:"data"`
}

type flowEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

func automationToYAML(a *sdk.AutomationRecord) ([]byte, error) {
	doc := automationDoc{
		Name:        a.Name,
		Description: a.Description,
		Sharing:     a.Sharing,
		Enabled:     a.Enabled,
		Nodes:       []map[string]any{},
		Edges:       []map[string]any{},
	}

	nodes, err := parseNodes(a.Nodes)
	if err != nil {
		return nil, fmt.Errorf("parse nodes: %w", err)
	}

	for _, n := range nodes {
		if n.Type == "trigger" {
			doc.Trigger = triggerFromNode(n)
			break
		}
	}

	for _, n := range nodes {
		entry := map[string]any{
			"id":   n.ID,
			"type": n.Type,
		}
		if n.Position != nil {
			entry["position"] = n.Position
		}
		if len(n.Data) > 0 {
			entry["data"] = n.Data
		}
		doc.Nodes = append(doc.Nodes, entry)
	}

	edges, err := parseEdges(a.Edges)
	if err != nil {
		return nil, fmt.Errorf("parse edges: %w", err)
	}
	for _, e := range edges {
		entry := map[string]any{
			"from": e.Source,
			"to":   e.Target,
		}
		if e.SourceHandle != "" {
			entry["sourceHandle"] = e.SourceHandle
		}
		if e.TargetHandle != "" {
			entry["targetHandle"] = e.TargetHandle
		}
		doc.Edges = append(doc.Edges, entry)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close yaml encoder: %w", err)
	}
	return buf.Bytes(), nil
}

func parseNodes(raw string) ([]flowNode, error) {
	if raw == "" {
		return nil, nil
	}
	var nodes []flowNode
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func parseEdges(raw string) ([]flowEdge, error) {
	if raw == "" {
		return nil, nil
	}
	var edges []flowEdge
	if err := json.Unmarshal([]byte(raw), &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func triggerFromNode(n flowNode) map[string]any {
	out := map[string]any{}
	if t, ok := n.Data["triggerType"].(string); ok && t != "" {
		out["type"] = t
	} else {
		out["type"] = "manual"
	}
	if cron, ok := n.Data["cron"].(string); ok && cron != "" {
		out["cron"] = cron
	}
	return out
}