<script lang="ts">
	interface Props {
		name: string;
	}

	let { name }: Props = $props();

	const colors: Record<string, string> = {
		anthropic: 'badge--amber',
		openai: 'badge--emerald',
		google: 'badge--blue',
		ollama: 'badge--purple',
		bedrock: 'badge--orange',
		vertex: 'badge--cyan',
		openrouter: 'badge--pink'
	};

	const colorClass = $derived(colors[name.toLowerCase()] ?? 'badge--neutral');
</script>

<span class="badge {colorClass}">{name}</span>

<style lang="scss">
	.badge {
		@include badge;

		&--amber { background: #fef3c7; color: #92400e; }
		&--emerald { background: $success-100; color: $success-700; }
		&--blue { background: $info-100; color: $info-600; }
		&--purple { background: #ede9fe; color: #5b21b6; }
		&--orange { background: #ffedd5; color: #9a3412; }
		&--cyan { background: #cffafe; color: #155e75; }
		&--pink { background: #fce7f3; color: #9d174d; }
		&--neutral { background: $neutral-100; color: $neutral-700; }
	}
</style>
