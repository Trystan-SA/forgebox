package brain

import (
	"regexp"
	"strings"
)

var (
	linkPattern    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	hashtagPattern = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9_-]+)`)
)

// ExtractLinks finds all [[link]] references in markdown content.
// Returns deduplicated titles in order of first appearance.
func ExtractLinks(content string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		title := strings.TrimSpace(m[1])
		if title != "" && !seen[title] {
			seen[title] = true
			links = append(links, title)
		}
	}
	return links
}

// ExtractHashtags finds all #hashtag references in markdown content.
// Returns deduplicated, lowercase tags in order of first appearance.
// Ignores anchor links like [text](#heading).
func ExtractHashtags(content string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}
