package plugins

// providerLabels maps a provider factory key to the human-readable label
// used as both the display name and the registry key for DB-backed
// providers. Operators do not get to customize this — see specs/3.1.2.
//
// Keep this in sync with providerFactories.
var providerLabels = map[string]string{
	"anthropic":              "Anthropic",
	"anthropic-api":          "Anthropic (API)",
	"anthropic-subscription": "Anthropic (Subscription)",
	"openai":                 "OpenAI",
	"ollama":                 "Ollama",
}

// LabelForType returns the canonical display label for a provider factory
// key. Returns the second value `false` if the type is unknown.
func LabelForType(typ string) (string, bool) {
	l, ok := providerLabels[typ]
	return l, ok
}
