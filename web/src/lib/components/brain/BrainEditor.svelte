<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
	import { Editor, Mark } from '@tiptap/core';
	import StarterKit from '@tiptap/starter-kit';
	import Placeholder from '@tiptap/extension-placeholder';
	import type { BrainFile } from '$lib/api/types';

	const BrainLinkMark = Mark.create({
		name: 'brainLink',
		inclusive: false,
		addAttributes() {
			return {
				title: {
					default: null,
					parseHTML: (el) => (el as HTMLElement).getAttribute('data-link'),
					renderHTML: (attrs) =>
						attrs.title ? { 'data-link': attrs.title as string } : {}
				}
			};
		},
		parseHTML() {
			return [{ tag: 'span[data-brain-link]' }];
		},
		renderHTML({ HTMLAttributes }) {
			return [
				'span',
				{ ...HTMLAttributes, 'data-brain-link': '', class: 'brain-link-badge' },
				0
			];
		}
	});

	interface Props {
		file: BrainFile | null;
		allFiles: BrainFile[];
		allHashtags: string[];
	}

	let { file, allFiles, allHashtags }: Props = $props();

	const dispatch = createEventDispatcher<{
		save: { title: string; content: string };
		delete: Record<string, never>;
	}>();

	let editorContainer: HTMLDivElement;
	let editor: Editor | null = null;

	let autocomplete = $state<{
		type: 'file' | 'hashtag';
		query: string;
		items: string[];
		selectedIdx: number;
	} | null>(null);

	let saving = $state(false);

	function htmlToMarkdown(html: string): string {
		return html
			.replace(/<h1[^>]*>(.*?)<\/h1>/gi, '# $1\n')
			.replace(/<h2[^>]*>(.*?)<\/h2>/gi, '## $1\n')
			.replace(/<h3[^>]*>(.*?)<\/h3>/gi, '### $1\n')
			.replace(/<strong[^>]*>(.*?)<\/strong>/gi, '**$1**')
			.replace(/<em[^>]*>(.*?)<\/em>/gi, '_$1_')
			.replace(/<code[^>]*>(.*?)<\/code>/gi, '`$1`')
			.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, '- $1\n')
			.replace(/<br\s*\/?>/gi, '\n')
			.replace(/<\/p>\s*<p[^>]*>/gi, '\n\n')
			.replace(/<p[^>]*>([\s\S]*?)<\/p>/gi, '$1\n')
			.replace(/<span[^>]*\bdata-brain-link\b[^>]*>([\s\S]*?)<\/span>/gi, (match, inner) => {
				const linkAttr = match.match(/data-link="([^"]*)"/i);
				const title = linkAttr ? linkAttr[1] : inner;
				return `[[${title}]]`;
			})
			.replace(/<[^>]+>/g, '')
			.replace(/&amp;/g, '&')
			.replace(/&lt;/g, '<')
			.replace(/&gt;/g, '>')
			.replace(/&quot;/g, '"')
			.replace(/&#39;/g, "'")
			.trim();
	}

	function markdownToHtml(md: string): string {
		if (!md) return '';
		const paragraphs = md.split(/\n{2,}/);

		return paragraphs
			.map((para) => {
				let html = para
					.replace(/^### (.+)$/gm, '<h3>$1</h3>')
					.replace(/^## (.+)$/gm, '<h2>$1</h2>')
					.replace(/^# (.+)$/gm, '<h1>$1</h1>')
					.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
					.replace(/_(.+?)_/g, '<em>$1</em>')
					.replace(/`(.+?)`/g, '<code>$1</code>')
					.replace(/\[\[([^\[\]]+)\]\]/g, (_m, title) => {
						const safe = String(title).replace(/"/g, '&quot;');
						return `<span data-brain-link data-link="${safe}">${title}</span>`;
					})
					.replace(/^- (.+)$/gm, '<li>$1</li>')
					.replace(/\n/g, '<br>');

				html = html.replace(/(<li>.*<\/li>\s*)+/g, (m) => `<ul>${m}</ul>`);
				if (!html.trim()) return '';
				if (/^\s*<(h[1-6]|ul|ol|li|blockquote|pre)\b/i.test(html)) return html;
				return `<p>${html}</p>`;
			})
			.join('');
	}

	function destroyEditor() {
		if (editor) {
			editor.destroy();
			editor = null;
		}
	}

	function createEditor(content: string) {
		destroyEditor();
		if (!editorContainer) return;

		editor = new Editor({
			element: editorContainer,
			extensions: [
				StarterKit.configure({ link: { openOnClick: false } }),
				BrainLinkMark,
				Placeholder.configure({ placeholder: 'Write your notes here...' })
			],
			content: markdownToHtml(content),
			editorProps: {
				attributes: { class: 'brain-editor__tiptap' },
				handleClickOn: (view, _pos, _node, _nodePos, event) => {
					const target = event.target as HTMLElement | null;
					if (!target) return false;
					const badge = target.closest('span[data-brain-link]') as HTMLElement | null;
					if (!badge) return false;

					const rect = badge.getBoundingClientRect();
					const distFromRight = rect.right - (event as MouseEvent).clientX;
					if (distFromRight > 18 || distFromRight < 0) return false;

					const start = view.posAtDOM(badge, 0);
					const end = start + (badge.textContent ?? '').length;
					if (start < 0 || end <= start) return false;

					const tr = view.state.tr.delete(start, end);
					view.dispatch(tr);
					event.preventDefault();
					return true;
				}
			},
			onUpdate: ({ editor: e, transaction }) => {
				if (!transaction.getMeta('brainLinkSweep')) {
					sweepBrainLinks(e);
				}
				const cursor = e.state.selection.from;
				const textBeforeCursor = e.state.doc.textBetween(0, cursor, '\n', '\n');
				checkAutocomplete(textBeforeCursor);
			}
		});
	}

	function sweepBrainLinks(e: Editor) {
		const markType = e.schema.marks.brainLink;
		if (!markType) return;
		const tr = e.state.tr;
		let modified = false;
		e.state.doc.descendants((node, pos) => {
			if (!node.isText) return;
			const mark = node.marks.find((m) => m.type === markType);
			if (!mark) return;
			const expected = String(mark.attrs.title ?? '');
			if (node.text !== expected) {
				tr.removeMark(pos, pos + node.nodeSize, markType);
				modified = true;
			}
		});
		if (modified) {
			tr.setMeta('brainLinkSweep', true);
			tr.setMeta('addToHistory', false);
			e.view.dispatch(tr);
		}
	}

	function checkAutocomplete(text: string) {
		const lines = text.split('\n');
		const lastLine = lines[lines.length - 1] ?? '';

		const fileMatch = lastLine.match(/\[\[([^\]]{0,40})$/);
		if (fileMatch) {
			const query = fileMatch[1].toLowerCase();
			const items = allFiles
				.filter((f) => f.id !== file?.id && f.title.toLowerCase().includes(query))
				.map((f) => f.title)
				.slice(0, 8);
			autocomplete = { type: 'file', query: fileMatch[1], items, selectedIdx: 0 };
			return;
		}

		const hashMatch = lastLine.match(/#([a-zA-Z0-9_-]{0,30})$/);
		if (hashMatch) {
			const query = hashMatch[1].toLowerCase();
			const items = allHashtags.filter((h) => h.toLowerCase().includes(query)).slice(0, 8);
			autocomplete = { type: 'hashtag', query: hashMatch[1], items, selectedIdx: 0 };
			return;
		}

		autocomplete = null;
	}

	function insertAutocompleteItem(item: string) {
		if (!editor || !autocomplete) return;

		const { from } = editor.state.selection;
		const prefix = autocomplete.type === 'file' ? '[[' : '#';
		const queryLen = autocomplete.query.length + prefix.length;
		const docSize = editor.state.doc.content.size;
		const trailingCloser =
			autocomplete.type === 'file' &&
			editor.state.doc.textBetween(from, Math.min(from + 2, docSize)) === ']]'
				? 2
				: 0;

		if (autocomplete.type === 'file') {
			const safeTitle = item.replace(/"/g, '&quot;');
			const html = `<span data-brain-link data-link="${safeTitle}">${item}</span>&nbsp;`;
			editor
				.chain()
				.focus()
				.deleteRange({ from: from - queryLen, to: from + trailingCloser })
				.insertContent(html)
				.run();
		} else {
			editor
				.chain()
				.focus()
				.deleteRange({ from: from - queryLen, to: from })
				.insertContent(`#${item} `)
				.run();
		}

		autocomplete = null;
	}

	function handleEditorKeyDown(e: KeyboardEvent) {
		if (!autocomplete || autocomplete.items.length === 0) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			autocomplete = {
				...autocomplete,
				selectedIdx: (autocomplete.selectedIdx + 1) % autocomplete.items.length
			};
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			autocomplete = {
				...autocomplete,
				selectedIdx:
					(autocomplete.selectedIdx - 1 + autocomplete.items.length) % autocomplete.items.length
			};
		} else if (e.key === 'Enter' || e.key === 'Tab') {
			e.preventDefault();
			const item = autocomplete.items[autocomplete.selectedIdx];
			if (item) insertAutocompleteItem(item);
		} else if (e.key === 'Escape') {
			autocomplete = null;
		}
	}

	function handleSave() {
		if (!editor || !file) return;
		const content = htmlToMarkdown(editor.getHTML());
		dispatch('save', { title: file.title, content });
	}

	function handleDelete() {
		if (!file) return;
		dispatch('delete', {});
	}

	function setFormat(format: string) {
		if (!editor) return;
		switch (format) {
			case 'bold': editor.chain().focus().toggleBold().run(); break;
			case 'italic': editor.chain().focus().toggleItalic().run(); break;
			case 'h1': editor.chain().focus().toggleHeading({ level: 1 }).run(); break;
			case 'h2': editor.chain().focus().toggleHeading({ level: 2 }).run(); break;
			case 'h3': editor.chain().focus().toggleHeading({ level: 3 }).run(); break;
			case 'bullet': editor.chain().focus().toggleBulletList().run(); break;
			case 'ordered': editor.chain().focus().toggleOrderedList().run(); break;
			case 'code': editor.chain().focus().toggleCodeBlock().run(); break;
		}
	}

	onMount(() => {
		if (file) createEditor(file.content ?? '');
	});

	onDestroy(() => {
		destroyEditor();
	});

	$effect(() => {
		if (file) {
			createEditor(file.content ?? '');
		} else {
			destroyEditor();
		}
	});

	function isActive(format: string): boolean {
		if (!editor) return false;
		switch (format) {
			case 'bold': return editor.isActive('bold');
			case 'italic': return editor.isActive('italic');
			case 'h1': return editor.isActive('heading', { level: 1 });
			case 'h2': return editor.isActive('heading', { level: 2 });
			case 'h3': return editor.isActive('heading', { level: 3 });
			case 'bullet': return editor.isActive('bulletList');
			case 'ordered': return editor.isActive('orderedList');
			case 'code': return editor.isActive('codeBlock');
			default: return false;
		}
	}
</script>

<div class="brain-editor">
	{#if !file}
		<div class="brain-editor__placeholder">
			<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
				<polyline points="14 2 14 8 20 8" />
				<line x1="16" y1="13" x2="8" y2="13" />
				<line x1="16" y1="17" x2="8" y2="17" />
				<polyline points="10 9 9 9 8 9" />
			</svg>
			<p>Select a file or create one</p>
		</div>
	{:else}
		<div class="brain-editor__toolbar">
			{#each [
				{ key: 'bold', label: 'B', title: 'Bold' },
				{ key: 'italic', label: 'I', title: 'Italic' },
				{ key: 'h1', label: 'H1', title: 'Heading 1' },
				{ key: 'h2', label: 'H2', title: 'Heading 2' },
				{ key: 'h3', label: 'H3', title: 'Heading 3' },
				{ key: 'bullet', label: '•—', title: 'Bullet List' },
				{ key: 'ordered', label: '1.', title: 'Numbered List' },
				{ key: 'code', label: '</>', title: 'Code Block' }
			] as btn}
				<button
					type="button"
					class="brain-editor__tb-btn"
					class:brain-editor__tb-btn--active={isActive(btn.key)}
					onclick={() => setFormat(btn.key)}
					title={btn.title}
				>
					{btn.label}
				</button>
			{/each}

			<div class="brain-editor__toolbar-spacer"></div>

			<button type="button" class="brain-editor__save btn-primary" onclick={handleSave} disabled={saving}>
				{saving ? 'Saving…' : 'Save'}
			</button>
			<button type="button" class="brain-editor__delete btn-danger" onclick={handleDelete}>
				Delete
			</button>
		</div>

		<div class="brain-editor__body" onkeydown={handleEditorKeyDown} role="textbox" tabindex="-1">
			<div bind:this={editorContainer} class="brain-editor__content"></div>

			{#if autocomplete && autocomplete.items.length > 0}
				<div class="brain-editor__autocomplete">
					<div class="brain-editor__autocomplete-label">
						{autocomplete.type === 'file' ? 'Link to file' : 'Hashtag'}
					</div>
					{#each autocomplete.items as item, i}
						<button
							type="button"
							class="brain-editor__autocomplete-item"
							class:brain-editor__autocomplete-item--selected={i === autocomplete.selectedIdx}
							onclick={() => insertAutocompleteItem(item)}
						>
							{autocomplete.type === 'file' ? `[[${item}]]` : `#${item}`}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style lang="scss">
	.brain-editor {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: $neutral-0;
		border-radius: $radius-xl;
		border: 1px solid $neutral-200;
		overflow: hidden;

		&__placeholder {
			@include flex-center;
			flex-direction: column;
			gap: $space-3;
			height: 100%;
			color: $neutral-400;
			font-size: $text-sm;

			p { margin: 0; }
		}

		&__toolbar {
			display: flex;
			align-items: center;
			gap: 2px;
			padding: $space-2 $space-3;
			border-bottom: 1px solid $neutral-200;
			background: $neutral-50;
			flex-wrap: wrap;
		}

		&__tb-btn {
			@include btn;
			padding: 4px 8px;
			font-size: $text-xs;
			font-weight: $font-semibold;
			font-family: $font-mono;
			color: $neutral-600;
			background: transparent;
			border-radius: $radius-md;
			min-width: 28px;

			&:hover { background: $neutral-200; color: $neutral-800; }

			&--active {
				background: $primary-100;
				color: $primary-700;
			}
		}

		&__toolbar-spacer { flex: 1; }

		&__save {
			@include btn;
			padding: 4px 12px;
			font-size: $text-xs;
			font-weight: $font-semibold;
			background: $primary-600;
			color: $neutral-0;
			border-radius: $radius-md;

			&:hover:not(:disabled) { background: $primary-700; }
		}

		&__delete {
			@include btn;
			padding: 4px 12px;
			font-size: $text-xs;
			font-weight: $font-semibold;
			background: $error-600;
			color: $neutral-0;
			border-radius: $radius-md;

			&:hover { background: $error-700; }
		}

		&__body {
			position: relative;
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
		}

		&__content {
			padding: $space-5;
			min-height: 100%;

			:global(.brain-editor__tiptap) {
				outline: none;
				min-height: 200px;
				font-size: $text-sm;
				line-height: $leading-relaxed;
				color: $neutral-800;

				:global(h1) { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; margin-bottom: $space-3; }
				:global(h2) { font-size: $text-lg; font-weight: $font-semibold; color: $neutral-900; margin-bottom: $space-2; }
				:global(h3) { font-size: $text-base; font-weight: $font-semibold; color: $neutral-800; margin-bottom: $space-2; }
				:global(p) { margin-bottom: $space-2; }
				:global(ul), :global(ol) { padding-left: $space-5; margin-bottom: $space-2; }
				:global(li) { margin-bottom: $space-1; }
				:global(code) { font-family: $font-mono; font-size: $text-xs; background: $neutral-100; padding: 2px 5px; border-radius: $radius-sm; }
				:global(pre) { background: $neutral-900; color: $neutral-100; padding: $space-4; border-radius: $radius-lg; margin-bottom: $space-3; overflow-x: auto; }
				:global(strong) { font-weight: $font-semibold; }
				:global(em) { font-style: italic; }
				:global(span[data-brain-link]) {
					display: inline-flex;
					align-items: center;
					gap: 4px;
					padding: 1px 4px 1px 8px;
					font-family: $font-sans;
					font-size: $text-xs;
					font-weight: $font-medium;
					color: #6d28d9;
					background: #f5f3ff;
					border: 1px solid #ddd6fe;
					border-radius: $radius-md;
					line-height: 1.4;
					white-space: nowrap;
					cursor: default;
					transition: background 0.15s ease, border-color 0.15s ease;

					&::before {
						content: '';
						display: inline-block;
						width: 6px;
						height: 6px;
						border-radius: 50%;
						background: #8b5cf6;
					}

					&::after {
						content: '×';
						display: inline-flex;
						align-items: center;
						justify-content: center;
						width: 14px;
						height: 14px;
						margin-left: 2px;
						font-size: 13px;
						line-height: 1;
						font-weight: $font-bold;
						color: #a78bfa;
						border-radius: 999px;
						cursor: pointer;
						transition: background 0.15s ease, color 0.15s ease;
					}

					&:hover {
						background: #ede9fe;
						border-color: #c4b5fd;
					}

					&:hover::after {
						background: #c4b5fd;
						color: #4c1d95;
					}
				}
				:global(.is-editor-empty:first-child::before) {
					content: attr(data-placeholder);
					color: $neutral-400;
					pointer-events: none;
					float: left;
					height: 0;
				}
			}
		}

		&__autocomplete {
			position: absolute;
			bottom: $space-4;
			left: $space-5;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			box-shadow: $shadow-md;
			z-index: 30;
			min-width: 200px;
			overflow: hidden;
		}

		&__autocomplete-label {
			font-size: 10px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.06em;
			color: $neutral-400;
			padding: $space-2 $space-3 $space-1;
		}

		&__autocomplete-item {
			display: block;
			width: 100%;
			text-align: left;
			padding: $space-2 $space-3;
			font-size: $text-xs;
			font-family: $font-mono;
			color: $neutral-700;
			background: transparent;
			border: none;
			cursor: pointer;
			transition: background $transition-fast;

			&:hover, &--selected {
				background: $primary-50;
				color: $primary-700;
			}
		}
	}
</style>
