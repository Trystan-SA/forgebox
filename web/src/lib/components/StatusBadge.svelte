<script lang="ts">
	import type { TaskStatus } from '$lib/api/types';

	interface Props {
		status: TaskStatus;
	}

	let { status }: Props = $props();

	const config: Record<TaskStatus, { label: string; className: string }> = {
		pending: { label: 'Pending', className: 'badge--neutral' },
		running: { label: 'Running', className: 'badge--info' },
		completed: { label: 'Completed', className: 'badge--success' },
		failed: { label: 'Failed', className: 'badge--error' },
		cancelled: { label: 'Cancelled', className: 'badge--warning' }
	};

	const current = $derived(config[status] ?? config.pending);
</script>

<span class="badge {current.className}" class:badge--animated={status === 'running'}>
	{current.label}
</span>

<style lang="scss">
	.badge {
		@include badge;

		&--neutral { background: $neutral-100; color: $neutral-700; }
		&--info { background: $info-100; color: $info-600; }
		&--success { background: $success-100; color: $success-700; }
		&--error { background: $error-100; color: $error-700; }
		&--warning { background: $warning-100; color: $warning-700; }

		&--animated::before {
			content: '';
			width: 6px;
			height: 6px;
			border-radius: 50%;
			background: currentColor;
			animation: pulse 1.5s ease-in-out infinite;
		}
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
</style>
