import type { Provider, ProviderType } from '$lib/api/types';

// Canonical display label for a provider type. Mirrors the server-side
// table in internal/plugins/labels.go — keep them in sync. Per specs/3.1.2
// operators don't customize the display name, so anything we render uses
// the type-derived label.
const PROVIDER_TYPE_LABELS: Record<ProviderType, string> = {
	anthropic: 'Anthropic',
	'anthropic-api': 'Anthropic (API)',
	'anthropic-subscription': 'Anthropic (Subscription)',
	openai: 'OpenAI',
	ollama: 'Ollama'
};

export function providerLabel(p: Provider): string {
	if (p.provider_type && PROVIDER_TYPE_LABELS[p.provider_type]) {
		return PROVIDER_TYPE_LABELS[p.provider_type];
	}
	return p.name;
}
