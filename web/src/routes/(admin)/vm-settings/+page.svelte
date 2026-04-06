<script lang="ts">
	const vmDefaults = [
		{ label: 'Memory', value: '512 MB' },
		{ label: 'vCPUs', value: '1' },
		{ label: 'Timeout', value: '5 minutes' },
		{ label: 'Network Access', value: 'Disabled' }
	];
</script>

<div class="page">
	<div class="page__header">
		<h1>VM Settings</h1>
		<p>Firecracker VM defaults, network policies, and resource limits</p>
	</div>

	<h2>VM Defaults</h2>
	<div class="settings-card">
		{#each vmDefaults as item}
			<div class="settings-row">
				<span class="settings-row__label">{item.label}</span>
				<span class="settings-row__value">{item.value}</span>
			</div>
		{/each}
	</div>

	<h2 class="section-title">Storage</h2>
	<div class="settings-card">
		<div class="settings-info">
			Local filesystem storage is active. Configure S3 or GCS backends in the
			ForgeBox configuration file.
		</div>
	</div>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.section-title { margin-top: $space-8; margin-bottom: $space-4; }

	.settings-card {
		@include card;
		margin-top: $space-4;
	}

	.settings-row {
		@include flex-between;
		padding: $space-3 $space-5;
		font-size: $text-sm;

		& + & { border-top: 1px solid $neutral-100; }

		&__label { color: $neutral-600; }
		&__value { font-weight: $font-medium; color: $neutral-900; }
	}

	.settings-info {
		padding: $space-4 $space-5;
		font-size: $text-sm;
		color: $neutral-600;
	}
</style>
