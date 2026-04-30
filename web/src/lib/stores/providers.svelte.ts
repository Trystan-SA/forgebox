import { listProviders } from '$lib/api/client';
import type { Provider } from '$lib/api/types';

// Shared provider list cache. The app layout reads this for the sidebar's
// "no providers configured" indicator; the providers pages refresh() after
// create/delete so the indicator updates without a per-navigation refetch.
type State = {
	providers: Provider[];
	loaded: boolean;
};

export const providersStore = $state<State>({ providers: [], loaded: false });

let inflight: Promise<Provider[]> | null = null;

export async function loadProviders(): Promise<Provider[]> {
	if (inflight) return inflight;
	inflight = listProviders()
		.then((list) => {
			providersStore.providers = list;
			providersStore.loaded = true;
			return list;
		})
		.finally(() => {
			inflight = null;
		});
	return inflight;
}

export async function refreshProviders(): Promise<Provider[]> {
	inflight = null;
	return loadProviders();
}
