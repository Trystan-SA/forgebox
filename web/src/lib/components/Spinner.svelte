<script lang="ts">
	interface Props {
		size?: 'sm' | 'md' | 'lg';
	}

	let { size = 'md' }: Props = $props();
</script>

<svg
	class="spinner spinner--{size}"
	viewBox="0 0 24 24"
	fill="none"
	xmlns="http://www.w3.org/2000/svg"
>
	<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" opacity="0.25" />
	<path
		d="M12 2a10 10 0 0 1 10 10"
		stroke="currentColor"
		stroke-width="3"
		stroke-linecap="round"
	/>
</svg>

<style lang="scss">
	.spinner {
		animation: spin 0.8s linear infinite;

		&--sm { width: 1rem; height: 1rem; }
		&--md { width: 1.5rem; height: 1.5rem; }
		&--lg { width: 2rem; height: 2rem; }
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
