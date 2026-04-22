package brain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single link",
			content: "See [[Project Setup]] for details.",
			want:    []string{"Project Setup"},
		},
		{
			name:    "multiple links",
			content: "Check [[Auth Guide]] and [[API Design]].",
			want:    []string{"Auth Guide", "API Design"},
		},
		{
			name:    "no links",
			content: "Plain markdown with no links.",
			want:    nil,
		},
		{
			name:    "link at start",
			content: "[[Quick Start]] is the first step.",
			want:    []string{"Quick Start"},
		},
		{
			name:    "duplicate links deduplicated",
			content: "See [[Auth]] and also [[Auth]] again.",
			want:    []string{"Auth"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLinks(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractHashtags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single tag",
			content: "This is about #deployment.",
			want:    []string{"deployment"},
		},
		{
			name:    "multiple tags",
			content: "#auth and #security are related.",
			want:    []string{"auth", "security"},
		},
		{
			name:    "tag at line start",
			content: "#setup\nSome content.",
			want:    []string{"setup"},
		},
		{
			name:    "no tags",
			content: "Plain markdown content.",
			want:    nil,
		},
		{
			name:    "normalizes to lowercase",
			content: "About #DevOps and #CI-CD.",
			want:    []string{"devops", "ci-cd"},
		},
		{
			name:    "ignores anchors in links",
			content: "See [link](#heading) for info.",
			want:    nil,
		},
		{
			name:    "duplicates deduplicated",
			content: "#auth and #auth again.",
			want:    []string{"auth"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHashtags(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
