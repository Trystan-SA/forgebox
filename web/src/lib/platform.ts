export function isTauri(): boolean {
	return typeof window !== 'undefined' && '__TAURI__' in window;
}

export function getBaseUrl(): string {
	if (isTauri()) {
		const stored = localStorage.getItem('forgebox_api_url');
		return stored || 'http://localhost:8420/api/v1';
	}
	return '/api/v1';
}
