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
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Sharing     string         `yaml:"sharing"`
	Enabled     bool           `yaml:"enabled"`
	Trigger     map[string]any `yaml:"trigger,omitempty"`
	Nodes       []nodeOut      `yaml:"nodes"`
	Edges       []edgeOut      `yaml:"edges"`
}

// nodeOut serializes a canvas node to YAML. Position is emitted as a flow-style
// [x, y] sequence (not {x: _, y: _}) because yaml.v3 double-quotes the key
// "y" for YAML 1.1 boolean compatibility (y/n/yes/no/on/off are booleans in
// 1.1), which hurts readability.
type nodeOut struct {
	ID       string         `yaml:"id"`
	Type     string         `yaml:"type"`
	Position []int          `yaml:"position,flow,omitempty"`
	Data     map[string]any `yaml:"data,omitempty"`
}

type edgeOut struct {
	From         string `yaml:"from"`
	To           string `yaml:"to"`
	SourceHandle string `yaml:"sourceHandle,omitempty"`
	TargetHandle string `yaml:"targetHandle,omitempty"`
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
		Nodes:       []nodeOut{},
		Edges:       []edgeOut{},
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
		doc.Nodes = append(doc.Nodes, nodeOut{
			ID:       n.ID,
			Type:     n.Type,
			Position: positionXY(n.Position),
			Data:     n.Data,
		})
	}

	edges, err := parseEdges(a.Edges)
	if err != nil {
		return nil, fmt.Errorf("parse edges: %w", err)
	}
	for _, e := range edges {
		doc.Edges = append(doc.Edges, edgeOut{
			From:         e.Source,
			To:           e.Target,
			SourceHandle: e.SourceHandle,
			TargetHandle: e.TargetHandle,
		})
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

func positionXY(p map[string]any) []int {
	if p == nil {
		return nil
	}
	x, okX := toFloat(p["x"])
	y, okY := toFloat(p["y"])
	if !okX && !okY {
		return nil
	}
	return []int{int(x), int(y)}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
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