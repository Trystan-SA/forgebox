// Domain-level automation actions. These sit above the raw REST client and
// bundle the API call together with user-facing feedback (toasts) so pages
// don't re-implement the same save + success/error flow.
//
// Return value: the updated Automation on success, or null on failure. Pages
// should treat null as "already surfaced an error toast, do not retry here."
import type { Automation } from './types';
import type { Node, Edge } from '@xyflow/svelte';
import { updateAutomation } from './client';
import { pushToast } from '$lib/stores/toasts.svelte';

export async function saveAutomationGraph(
	id: string,
	nodes: Node[],
	edges: Edge[]
): Promise<Automation | null> {
	try {
		const updated = await updateAutomation(id, {
			nodes: JSON.stringify(nodes),
			edges: JSON.stringify(edges)
		});
		pushToast('Automation saved', 'success');
		return updated;
	} catch (err) {
		pushToast(err instanceof Error ? err.message : 'Save failed', 'error', 5000);
		return null;
	}
}

export async function saveAutomationMeta(
	id: string,
	patch: { name: string; description: string }
): Promise<Automation | null> {
	try {
		const updated = await updateAutomation(id, patch);
		pushToast('Details updated', 'success');
		return updated;
	} catch (err) {
		pushToast(err instanceof Error ? err.message : 'Update failed', 'error', 5000);
		return null;
	}
}