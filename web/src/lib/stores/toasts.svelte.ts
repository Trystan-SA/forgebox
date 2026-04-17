// Global toast queue used for transient user feedback (save confirmations,
// non-fatal errors). Mount <Toasts /> once in the app layout to render them;
// call `pushToast` from anywhere to enqueue a message.
export type ToastKind = 'success' | 'error' | 'info';

export interface Toast {
	id: number;
	message: string;
	kind: ToastKind;
}

export const toasts = $state<Toast[]>([]);

let nextId = 0;

export function pushToast(message: string, kind: ToastKind = 'info', duration = 3000): number {
	const id = ++nextId;
	toasts.push({ id, message, kind });
	if (duration > 0) {
		setTimeout(() => dismissToast(id), duration);
	}
	return id;
}

export function dismissToast(id: number): void {
	const idx = toasts.findIndex((t) => t.id === id);
	if (idx >= 0) toasts.splice(idx, 1);
}