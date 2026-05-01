<script lang="ts">
	import type { Node } from '@xyflow/svelte';
	import { onMount } from 'svelte';
	import { loadProviders, providersStore } from '$lib/stores/providers.svelte';
	import ModelSelector from '$lib/components/ModelSelector.svelte';

	interface Props {
		node: Node;
		onupdate: (id: string, data: Record<string, any>) => void;
		onclose: () => void;
	}

	let { node, onupdate, onclose }: Props = $props();

	let data = $state<Record<string, any>>({});

	$effect(() => {
		data = { ...node.data };
	});

	function update(field: string, value: any) {
		data[field] = value;
		onupdate(node.id, { ...data });
	}

	// Mirror data.provider / data.model into local state so we can use
	// ModelSelector's bind:value pattern, then push changes back via update().
	let aiProvider = $state('');
	let aiModel = $state('');

	$effect(() => {
		aiProvider = (data.provider as string) ?? '';
		aiModel = (data.model as string) ?? '';
	});

	$effect(() => {
		if (aiProvider !== (data.provider ?? '')) update('provider', aiProvider);
	});
	$effect(() => {
		if (aiModel !== (data.model ?? '')) update('model', aiModel);
	});

	onMount(() => {
		void loadProviders();
	});

	let panelWidth = $state(320);
	let resizing = $state(false);

	function startResize(e: MouseEvent) {
		e.preventDefault();
		resizing = true;
		const startX = e.clientX;
		const startWidth = panelWidth;

		function onMove(ev: MouseEvent) {
			const delta = startX - ev.clientX;
			panelWidth = Math.max(260, Math.min(600, startWidth + delta));
		}

		function onUp() {
			resizing = false;
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
		}

		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
	}

	let cronExpanded = $state(false);

	const schedFreq = $derived(data.schedFreq ?? 'hourly');
	const schedInterval = $derived(data.schedInterval ?? '1');
	const schedTime = $derived(data.schedTime ?? '09:00');
	const schedDays = $derived<string[]>(data.schedDays ?? []);

	const dayNames = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
	const dayValues = ['1', '2', '3', '4', '5', '6', '0'];

	function toggleDay(val: string) {
		const current = [...schedDays];
		const idx = current.indexOf(val);
		if (idx >= 0) current.splice(idx, 1);
		else current.push(val);
		update('schedDays', current);
		rebuildCron();
	}

	function setFreq(freq: string) {
		update('schedFreq', freq);
		rebuildCron(freq);
	}

	function setInterval(val: string) {
		update('schedInterval', val);
		rebuildCron();
	}

	function setTime(val: string) {
		update('schedTime', val);
		rebuildCron();
	}

	function rebuildCron(freq?: string) {
		const f = freq ?? data.schedFreq ?? 'hourly';
		const interval = data.schedInterval ?? '1';
		const time = data.schedTime ?? '09:00';
		const [h, m] = time.split(':').map(Number);
		const days = data.schedDays ?? [];

		let cron = '* * * * *';
		if (f === 'minutes') {
			cron = `*/${interval} * * * *`;
		} else if (f === 'hourly') {
			const iv = Number(interval);
			cron = iv > 1 ? `${m} */${iv} * * *` : `${m} * * * *`;
		} else if (f === 'daily') {
			cron = `${m} ${h} * * *`;
		} else if (f === 'weekly') {
			const d = days.length > 0 ? days.join(',') : '*';
			cron = `${m} ${h} * * ${d}`;
		} else if (f === 'monthly') {
			const d = interval || '1';
			cron = `${m} ${h} ${d} * *`;
		}
		update('cron', cron);
	}

	function setCronDirect(val: string) {
		update('cron', val);
		parseCronToFields(val);
	}

	function parseCronToFields(cron: string) {
		const p = cron.trim().split(/\s+/);
		if (p.length !== 5) return;

		const [mn, hr, dom, , dow] = p;

		if (mn.startsWith('*/') && hr === '*' && dom === '*' && dow === '*') {
			update('schedFreq', 'minutes');
			update('schedInterval', mn.slice(2));
			return;
		}

		if (/^\d+$/.test(mn) && hr.startsWith('*/') && dom === '*' && dow === '*') {
			update('schedFreq', 'hourly');
			update('schedInterval', hr.slice(2));
			update('schedTime', `00:${mn.padStart(2, '0')}`);
			return;
		}

		if (/^\d+$/.test(mn) && /^\d+$/.test(hr) && dom === '*' && dow !== '*') {
			update('schedFreq', 'weekly');
			update('schedTime', `${hr.padStart(2, '0')}:${mn.padStart(2, '0')}`);
			update('schedDays', dow.split(',').filter(Boolean));
			return;
		}

		if (/^\d+$/.test(mn) && /^\d+$/.test(hr) && /^\d+$/.test(dom) && dow === '*') {
			update('schedFreq', 'monthly');
			update('schedInterval', dom);
			update('schedTime', `${hr.padStart(2, '0')}:${mn.padStart(2, '0')}`);
			return;
		}

		if (/^\d+$/.test(mn) && /^\d+$/.test(hr) && dom === '*' && dow === '*') {
			update('schedFreq', 'daily');
			update('schedTime', `${hr.padStart(2, '0')}:${mn.padStart(2, '0')}`);
			return;
		}

		if (/^\d+$/.test(mn) && hr === '*' && dom === '*' && dow === '*') {
			update('schedFreq', 'hourly');
			update('schedInterval', '1');
			update('schedTime', `00:${mn.padStart(2, '0')}`);
			return;
		}
	}

	const cronParts = $derived((data.cron ?? '* * * * *').split(' '));
	const cronFields = ['min', 'hour', 'day', 'month', 'wday'] as const;

	const schedSummary = $derived.by(() => {
		const f = data.schedFreq ?? 'hourly';
		const interval = data.schedInterval ?? '1';
		const time = data.schedTime ?? '09:00';
		const days = data.schedDays ?? [];

		if (f === 'minutes') return `Every ${interval} minute${interval !== '1' ? 's' : ''}`;
		if (f === 'hourly') return Number(interval) > 1 ? `Every ${interval} hours at :${time.split(':')[1]}` : `Every hour at :${time.split(':')[1]}`;
		if (f === 'daily') return `Daily at ${time}`;
		if (f === 'weekly') {
			const d = days.map((v: string) => dayNames[dayValues.indexOf(v)]).filter(Boolean).join(', ');
			return d ? `Weekly on ${d} at ${time}` : `Weekly at ${time}`;
		}
		if (f === 'monthly') return `Monthly on day ${interval} at ${time}`;
		return 'Custom schedule';
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
	}

	const typeLabels: Record<string, string> = {
		trigger: 'Trigger',
		aiStep: 'AI Step',
		tool: 'Tool',
		condition: 'Condition',
		switch: 'Switch'
	};

	const operatorOptions: { value: string; label: string; types: string[] }[] = [
		{ value: 'equals', label: 'Equals (==)', types: ['string', 'number'] },
		{ value: 'not_equals', label: 'Not equals (!=)', types: ['string', 'number'] },
		{ value: 'contains', label: 'Contains', types: ['string'] },
		{ value: 'not_contains', label: 'Does not contain', types: ['string'] },
		{ value: 'gt', label: 'Greater than (>)', types: ['number'] },
		{ value: 'gte', label: 'Greater or equal (>=)', types: ['number'] },
		{ value: 'lt', label: 'Less than (<)', types: ['number'] },
		{ value: 'lte', label: 'Less or equal (<=)', types: ['number'] },
		{ value: 'is_true', label: 'Is true', types: ['boolean'] },
		{ value: 'is_false', label: 'Is false', types: ['boolean'] },
		{ value: 'is_empty', label: 'Is empty', types: ['string'] },
		{ value: 'is_not_empty', label: 'Is not empty', types: ['string'] }
	];

	const filteredOperators = $derived(
		operatorOptions.filter((op) => !data.valueType || op.types.includes(data.valueType))
	);

	const unaryOps = ['is_true', 'is_false', 'is_empty', 'is_not_empty'];
	const showValue = $derived(!unaryOps.includes(data.operator ?? ''));

	function addCase() {
		const cases = [...(data.cases ?? []), `Case ${(data.cases?.length ?? 0) + 1}`];
		update('cases', cases);
	}

	function removeCase(index: number) {
		const cases = [...(data.cases ?? [])];
		cases.splice(index, 1);
		update('cases', cases);
	}

	function updateCase(index: number, value: string) {
		const cases = [...(data.cases ?? [])];
		cases[index] = value;
		update('cases', cases);
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="panel" class:panel--resizing={resizing} style="width: {panelWidth}px;">
	<div class="panel__resize" onmousedown={startResize}></div>
	<div class="panel__head">
		<div class="panel__type">{typeLabels[node.type ?? ''] ?? node.type}</div>
		<button class="panel__close" onclick={onclose}>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<div class="panel__body">
		<label class="panel__field">
			<span class="panel__label">Label</span>
			<input
				class="panel__input"
				type="text"
				value={data.label ?? ''}
				oninput={(e) => update('label', e.currentTarget.value)}
				placeholder="Node name"
			/>
		</label>

		{#if node.type === 'trigger' && data.triggerType === 'schedule'}
			<label class="panel__field">
				<span class="panel__label">Frequency</span>
				<select class="panel__select" value={schedFreq} onchange={(e) => setFreq(e.currentTarget.value)}>
					<option value="minutes">Every X minutes</option>
					<option value="hourly">Hourly</option>
					<option value="daily">Daily</option>
					<option value="weekly">Weekly</option>
					<option value="monthly">Monthly</option>
				</select>
			</label>

			{#if schedFreq === 'minutes'}
				<label class="panel__field">
					<span class="panel__label">Every</span>
					<div class="sched__row">
						<select class="panel__select" value={schedInterval} onchange={(e) => setInterval(e.currentTarget.value)}>
							{#each ['1','2','5','10','15','30'] as v}
								<option value={v}>{v}</option>
							{/each}
						</select>
						<span class="sched__suffix">minutes</span>
					</div>
				</label>
			{/if}

			{#if schedFreq === 'hourly'}
				<label class="panel__field">
					<span class="panel__label">Every</span>
					<div class="sched__row">
						<select class="panel__select" value={schedInterval} onchange={(e) => setInterval(e.currentTarget.value)}>
							{#each ['1','2','3','4','6','8','12'] as v}
								<option value={v}>{v}</option>
							{/each}
						</select>
						<span class="sched__suffix">hour{Number(schedInterval) > 1 ? 's' : ''}</span>
					</div>
				</label>
				<label class="panel__field">
					<span class="panel__label">At minute</span>
					<input class="panel__input" type="time" value={schedTime} oninput={(e) => setTime(e.currentTarget.value)} />
				</label>
			{/if}

			{#if schedFreq === 'daily'}
				<label class="panel__field">
					<span class="panel__label">Time</span>
					<input class="panel__input" type="time" value={schedTime} oninput={(e) => setTime(e.currentTarget.value)} />
				</label>
			{/if}

			{#if schedFreq === 'weekly'}
				<div class="panel__field">
					<span class="panel__label">Days</span>
					<div class="sched__days">
						{#each dayNames as name, i}
							<button
								class="sched__day"
								class:sched__day--on={schedDays.includes(dayValues[i])}
								onclick={() => toggleDay(dayValues[i])}
							>{name}</button>
						{/each}
					</div>
				</div>
				<label class="panel__field">
					<span class="panel__label">Time</span>
					<input class="panel__input" type="time" value={schedTime} oninput={(e) => setTime(e.currentTarget.value)} />
				</label>
			{/if}

			{#if schedFreq === 'monthly'}
				<label class="panel__field">
					<span class="panel__label">Day of month</span>
					<select class="panel__select" value={schedInterval} onchange={(e) => setInterval(e.currentTarget.value)}>
						{#each Array.from({ length: 28 }, (_, i) => String(i + 1)) as d}
							<option value={d}>{d}</option>
						{/each}
					</select>
				</label>
				<label class="panel__field">
					<span class="panel__label">Time</span>
					<input class="panel__input" type="time" value={schedTime} oninput={(e) => setTime(e.currentTarget.value)} />
				</label>
			{/if}

			<div class="sched__summary">{schedSummary}</div>
		{/if}

		{#if node.type === 'aiStep'}
			<div class="panel__field">
				<span class="panel__label">Model</span>
				<ModelSelector providers={providersStore.providers} bind:provider={aiProvider} bind:model={aiModel} compact />
			</div>
			<label class="panel__field">
				<span class="panel__label">Prompt</span>
				<textarea class="panel__textarea" value={data.prompt ?? ''} oninput={(e) => update('prompt', e.currentTarget.value)} placeholder="Enter prompt..." rows="5"></textarea>
			</label>
		{/if}

		{#if node.type === 'tool'}
			<label class="panel__field">
				<span class="panel__label">Tool</span>
				<select class="panel__select" value={data.tool ?? ''} onchange={(e) => update('tool', e.currentTarget.value)}>
					<option value="bash">Shell (bash)</option>
					<option value="web_fetch">HTTP Request</option>
					<option value="file_read">File Read</option>
					<option value="file_write">File Write</option>
				</select>
			</label>
		{/if}

		{#if node.type === 'condition'}
			<label class="panel__field">
				<span class="panel__label">Value type</span>
				<select class="panel__select" value={data.valueType ?? ''} onchange={(e) => update('valueType', e.currentTarget.value)}>
					<option value="">Any</option>
					<option value="boolean">Boolean</option>
					<option value="string">String</option>
					<option value="number">Number</option>
				</select>
			</label>
			<label class="panel__field">
				<span class="panel__label">Field</span>
				<input class="panel__input" type="text" value={data.field ?? ''} oninput={(e) => update('field', e.currentTarget.value)} placeholder="e.g. result.status" />
			</label>
			<label class="panel__field">
				<span class="panel__label">Operator</span>
				<select class="panel__select" value={data.operator ?? ''} onchange={(e) => update('operator', e.currentTarget.value)}>
					<option value="">Select...</option>
					{#each filteredOperators as op}
						<option value={op.value}>{op.label}</option>
					{/each}
				</select>
			</label>
			{#if showValue}
				<label class="panel__field">
					<span class="panel__label">Value</span>
					<input class="panel__input" type={data.valueType === 'number' ? 'number' : 'text'} value={data.value ?? ''} oninput={(e) => update('value', e.currentTarget.value)} placeholder="Compare to..." />
				</label>
			{/if}
		{/if}

		{#if node.type === 'switch'}
			<label class="panel__field">
				<span class="panel__label">Field</span>
				<input class="panel__input" type="text" value={data.field ?? ''} oninput={(e) => update('field', e.currentTarget.value)} placeholder="e.g. result.type" />
			</label>
			<div class="panel__field">
				<div class="panel__label-row">
					<span class="panel__label">Cases</span>
					<button class="panel__add" onclick={addCase}>+ Add</button>
				</div>
				<div class="panel__cases">
					{#each data.cases ?? [] as caseName, i}
						<div class="panel__case">
							<input
								class="panel__input panel__input--case"
								type="text"
								value={caseName}
								oninput={(e) => updateCase(i, e.currentTarget.value)}
							/>
							<button class="panel__case-rm" onclick={() => removeCase(i)}>
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
								</svg>
							</button>
						</div>
					{/each}
					<div class="panel__case-default">
						<span>default</span>
						<span class="panel__case-hint">Always present</span>
					</div>
				</div>
			</div>
		{/if}

		{#if node.type === 'trigger' && data.triggerType === 'schedule'}
			<div class="expert">
				<button class="expert__toggle" onclick={() => { cronExpanded = !cronExpanded; }}>
					<svg class="expert__arrow" class:expert__arrow--open={cronExpanded} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6" /></svg>
					<span>Expert</span>
					<code class="expert__cron-badge">{data.cron ?? '* * * * *'}</code>
				</button>
				{#if cronExpanded}
					<div class="expert__body">
						<div class="expert__grid">
							{#each cronFields as field, i}
								<label class="expert__cell">
									<span class="expert__cell-label">{field}</span>
									<input
										class="expert__cell-input"
										type="text"
										value={cronParts[i] ?? '*'}
										oninput={(e) => { const parts = [...cronParts]; parts[i] = e.currentTarget.value || '*'; setCronDirect(parts.join(' ')); }}
									/>
								</label>
							{/each}
						</div>
						<label class="expert__raw">
							<span class="expert__raw-label">Cron expression</span>
							<input class="panel__input" type="text" value={data.cron ?? '* * * * *'} oninput={(e) => setCronDirect(e.currentTarget.value)} />
						</label>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

<style lang="scss">
	.panel {
		height: 100%;
		background: $neutral-0;
		border-left: 1px solid $neutral-200;
		display: flex;
		flex-direction: column;
		flex-shrink: 0;
		animation: panel-in 0.15s ease-out;
		position: relative;

		&--resizing {
			user-select: none;
		}

		&__resize {
			position: absolute;
			left: -3px;
			top: 0;
			bottom: 0;
			width: 6px;
			cursor: col-resize;
			z-index: 5;

			&::after {
				content: '';
				position: absolute;
				left: 2px;
				top: 0;
				bottom: 0;
				width: 2px;
				background: transparent;
				transition: background $transition-fast;
			}

			&:hover::after {
				background: $primary-400;
			}
		}

		&__head {
			@include flex-between;
			padding: $space-3 $space-4;
			border-bottom: 1px solid $neutral-200;
		}

		&__type {
			font-family: $font-mono;
			font-size: $text-xs;
			font-weight: $font-bold;
			color: $neutral-500;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__close {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 28px;
			border: none;
			background: none;
			border-radius: $radius-lg;
			color: $neutral-400;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-600; }
		}

		&__body {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-4;
			display: flex;
			flex-direction: column;
			gap: $space-4;
		}

		&__field {
			display: flex;
			flex-direction: column;
			gap: $space-1;
		}

		&__label {
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
			text-transform: uppercase;
			letter-spacing: 0.04em;
		}

		&__label-row {
			display: flex;
			align-items: center;
			justify-content: space-between;
		}

		&__input {
			@include input-base;
			font-size: $text-sm;
			padding: $space-2 $space-3;
		}

		&__select {
			@include input-base;
			font-size: $text-sm;
			padding: $space-2 $space-3;
			cursor: pointer;
		}

		&__textarea {
			@include input-base;
			font-size: $text-sm;
			font-family: $font-mono;
			padding: $space-2 $space-3;
			resize: vertical;
			min-height: 80px;
		}

		&__add {
			font-family: $font-mono;
			font-size: $text-xs;
			color: $primary-600;
			background: none;
			border: none;
			cursor: pointer;
			padding: $space-1 $space-2;
			border-radius: $radius-sm;
			transition: all $transition-fast;

			&:hover { background: $primary-50; }
		}

		&__cases {
			display: flex;
			flex-direction: column;
			gap: $space-2;
			margin-top: $space-1;
		}

		&__case {
			display: flex;
			gap: $space-1;
			align-items: center;
		}

		&__input--case {
			flex: 1;
		}

		&__case-rm {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 28px;
			border: none;
			background: none;
			border-radius: $radius-lg;
			color: $neutral-300;
			cursor: pointer;
			flex-shrink: 0;
			transition: all $transition-fast;

			&:hover { background: $error-50; color: $error-500; }
		}

		&__case-default {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: $space-2 $space-3;
			background: $neutral-50;
			border-radius: $radius-lg;
			font-family: $font-mono;
			font-size: $text-sm;
			color: $neutral-500;
		}

		&__case-hint {
			font-size: $text-xs;
			color: $neutral-300;
			margin-left: auto;
		}
	}

	.sched__row {
		display: flex;
		align-items: center;
		gap: $space-2;

		.panel__select { flex: 1; }
	}

	.sched__suffix {
		font-size: $text-sm;
		color: $neutral-500;
		white-space: nowrap;
	}

	.sched__days {
		display: flex;
		gap: 4px;
	}

	.sched__day {
		flex: 1;
		padding: $space-1 0;
		font-family: $font-mono;
		font-size: 10px;
		font-weight: $font-semibold;
		text-align: center;
		border: 1px solid $neutral-200;
		border-radius: $radius-md;
		background: $neutral-0;
		color: $neutral-500;
		cursor: pointer;
		transition: all $transition-fast;

		&:hover { border-color: $primary-300; color: $neutral-700; }

		&--on {
			background: $primary-600;
			border-color: $primary-600;
			color: $neutral-0;

			&:hover { background: $primary-700; border-color: $primary-700; }
		}
	}

	.sched__summary {
		font-size: $text-sm;
		color: $primary-600;
		font-weight: $font-medium;
		padding: $space-2 $space-3;
		background: $primary-50;
		border-radius: $radius-lg;
		text-align: center;
	}

	.expert {
		margin-top: auto;
		border-top: 1px solid $neutral-100;
		padding-top: $space-3;

		&__toggle {
			display: flex;
			align-items: center;
			gap: $space-2;
			width: 100%;
			padding: $space-2;
			background: none;
			border: none;
			border-radius: $radius-md;
			cursor: pointer;
			font-size: $text-xs;
			font-weight: $font-semibold;
			color: $neutral-400;
			text-transform: uppercase;
			letter-spacing: 0.04em;
			transition: all $transition-fast;

			&:hover { background: $neutral-50; color: $neutral-600; }
		}

		&__arrow {
			transition: transform $transition-fast;
			&--open { transform: rotate(90deg); }
		}

		&__cron-badge {
			margin-left: auto;
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-medium;
			color: $neutral-400;
			background: $neutral-100;
			padding: 2px 6px;
			border-radius: $radius-sm;
		}

		&__body {
			display: flex;
			flex-direction: column;
			gap: $space-3;
			padding: $space-2 0;
		}

		&__grid {
			display: grid;
			grid-template-columns: repeat(5, 1fr);
			gap: 4px;
		}

		&__cell {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 2px;
		}

		&__cell-label {
			font-family: $font-mono;
			font-size: 9px;
			color: $neutral-400;
			text-transform: uppercase;
		}

		&__cell-input {
			width: 100%;
			text-align: center;
			font-family: $font-mono;
			font-size: $text-sm;
			padding: $space-1;
			color: $neutral-800;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-md;
			transition: border-color $transition-fast;

			&:focus {
				outline: none;
				border-color: $primary-500;
			}
		}

		&__raw {
			display: flex;
			flex-direction: column;
			gap: $space-1;
		}

		&__raw-label {
			font-size: $text-xs;
			color: $neutral-400;
		}
	}

	@keyframes panel-in {
		from { opacity: 0; transform: translateX(12px); }
		to { opacity: 1; transform: translateX(0); }
	}
</style>
