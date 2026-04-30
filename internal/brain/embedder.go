package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder computes vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimension() int
}

// OpenAIEmbedder calls the OpenAI embeddings API.
type OpenAIEmbedder struct {
	apiKey     string
	model      string
	dimension  int
	httpClient *http.Client
}

// NewOpenAIEmbedder creates an embedder using the OpenAI API.
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	dim := 1536
	if model == "text-embedding-3-large" {
		dim = 3072
	}
	return &OpenAIEmbedder{
		apiKey:     apiKey,
		model:      model,
		dimension:  dim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Dimension returns the vector length produced by this embedder.
func (e *OpenAIEmbedder) Dimension() int { return e.dimension }

// Embed calls the OpenAI embeddings API and returns a float32 vector.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model": e.model,
		"input": text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse embedding response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return result.Data[0].Embedding, nil
}

// MockEmbedder is a test double that returns a fixed-dimension vector.
type MockEmbedder struct {
	dim    int
	called int
}

// NewMockEmbedder creates a mock embedder for testing.
func NewMockEmbedder(dim int) *MockEmbedder {
	return &MockEmbedder{dim: dim}
}

// Dimension returns the configured vector length.
func (m *MockEmbedder) Dimension() int { return m.dim }

// Embed returns a deterministic vector derived from the text length.
func (m *MockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	m.called++
	vec := make([]float32, m.dim)
	// Simple deterministic embedding: hash-like based on text length
	for i := range vec {
		vec[i] = float32(len(text)+i) / float32(m.dim)
	}
	return vec, nil
}
