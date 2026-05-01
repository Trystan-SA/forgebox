<script lang="ts">
	const channels = [
		{ name: 'Slack', configured: false },
		{ name: 'Discord', configured: false },
		{ name: 'Webhook', configured: true },
		{ name: 'Email', configured: false }
	];
</script>

<section class="section">
	<h2>Channels</h2>
	<p class="section__hint">Configure input channels for receiving tasks</p>
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
</section>

<style lang="scss">
	.section {
		h2 { margin-bottom: $space-4; }

		&__hint {
			margin: -$space-2 0 $space-4;
			font-size: $text-sm;
			color: $neutral-500;
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
