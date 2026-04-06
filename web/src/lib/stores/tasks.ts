import { writable, derived } from 'svelte/store';
import type { Task, TaskEvent } from '$lib/api/types';
import {
	listTasks as apiListTasks,
	createTask as apiCreateTask,
	cancelTask as apiCancelTask,
	streamTask as apiStreamTask
} from '$lib/api/client';
import type { CreateTaskRequest } from '$lib/api/types';

export const tasks = writable<Task[]>([]);
export const tasksLoading = writable(false);
export const tasksError = writable<string | null>(null);

export async function fetchTasks() {
	tasksLoading.set(true);
	tasksError.set(null);
	try {
		const result = await apiListTasks();
		tasks.set(result);
	} catch (err) {
		tasksError.set(err instanceof Error ? err.message : 'Failed to fetch tasks');
	} finally {
		tasksLoading.set(false);
	}
}

export async function submitTask(
	req: CreateTaskRequest,
	onEvent: (event: TaskEvent) => void,
	onError?: (error: Error) => void
): Promise<{ taskId: string; stop: () => void }> {
	const res = await apiCreateTask(req);
	const stop = apiStreamTask(res.task_id, onEvent, onError);
	return { taskId: res.task_id, stop };
}

export async function cancelRunningTask(id: string): Promise<void> {
	await apiCancelTask(id);
}
