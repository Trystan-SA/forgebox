<script lang="ts">
	const channels = [
		{ name: 'Slack', configured: false },
		{ name: 'Discord', configured: false },
		{ name: 'Webhook', configured: true },
		{ name: 'Email', configured: false }
	];
</script>

<div class="page">
	<div class="page__header">
		<h1>Channels</h1>
		<p>Configure input channels for receiving tasks</p>
	</div>

	<div class="grid">
		{#each channels as channel}
			<div class="channel-card">
				<div>
					<p class="channel-card__name">{channel.name}</p>
					<p class="channel-card__meta">{channel.configured ? 'Configured' : 'Not configured'}</p>
				</div>
				<span class="channel-card__badge" class:channel-card__badge--active={channel.configured}>
					{channel.configured ? 'Active' : 'Inactive'}
				</span>
			</div>
		{/each}
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

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: $space-4;
	}

	.channel-card {
		@include card;
		@include flex-between;
		padding: $space-5;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; }

		&__badge {
			@include badge;
			background: $neutral-100;
			color: $neutral-500;

			&--active { background: $success-100; color: $success-700; }
		}
	}
</style>
