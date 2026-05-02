<script lang="ts">
	type Decision = 'approve' | 'deny';

	interface Props {
		toolName: string;
		inputArgs: Record<string, unknown>;
		approvalId: string;
		resolved: { approved: boolean } | null;
		onresolve: (decision: Decision, approvalId: string) => void;
	}

	let { toolName, inputArgs, approvalId, resolved, onresolve }: Props = $props();

	const entries = $derived(
		Object.entries(inputArgs).map(([key, value]) => ({
			key,
			val: typeof value === 'string' ? value : JSON.stringify(value)
		}))
	);

	function decide(decision: Decision) {
		onresolve(decision, approvalId);
	}
</script>

{#if resolved}
	<div class="approval-resolved" class:approval-resolved--approved={resolved.approved}>
		{#if resolved.approved}
			✓ Approved
		{:else}
			✗ Denied
		{/if}
		<code>{toolName}</code>
	</div>
{:else}
	<div class="approval-card" role="alert">
		<div class="approval-card__header">
			The assistant wants to run <code>{toolName}</code>
		</div>
		{#if entries.length > 0}
			<table class="approval-card__args">
				<tbody>
					{#each entries as { key, val } (key)}
						<tr>
							<th scope="row">{key}</th>
							<td>{val}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
		<div class="approval-card__actions">
			<button type="button" class="btn-primary" onclick={() => decide('approve')}>Approve</button>
			<button type="button" class="btn-danger" onclick={() => decide('deny')}>Deny</button>
		</div>
	</div>
{/if}

<style lang="scss">
	.approval-card {
		background: $warning-50;
		border: 1px solid $warning-100;
		border-radius: $radius-lg;
		padding: $space-4;
		margin: $space-2 0;
		color: $warning-700;

		&__header {
			font-size: $text-sm;
			font-weight: $font-semibold;
			margin-bottom: $space-3;

			code {
				font-family: $font-mono;
				background: $warning-100;
				padding: 0 $space-1;
				border-radius: $radius-sm;
			}
		}

		&__args {
			width: 100%;
			font-size: $text-sm;
			margin-bottom: $space-3;
			border-collapse: collapse;

			th {
				text-align: left;
				padding: $space-1 $space-2 $space-1 0;
				color: $neutral-600;
				font-weight: $font-medium;
				vertical-align: top;
				white-space: nowrap;
			}

			td {
				padding: $space-1 0;
				word-break: break-word;
				font-family: $font-mono;
				color: $neutral-800;
			}
		}

		&__actions {
			display: flex;
			gap: $space-2;
		}
	}

	.approval-resolved {
		font-size: $text-sm;
		color: $error-700;
		margin: $space-1 0;

		code {
			font-family: $font-mono;
			background: $neutral-100;
			padding: 0 $space-1;
			border-radius: $radius-sm;
		}

		&--approved {
			color: $success-700;
		}
	}
</style>
