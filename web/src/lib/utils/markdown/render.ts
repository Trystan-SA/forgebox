import type { BlockNode, InlineNode } from './tokens';

const HTML_ESCAPES: Record<string, string> = {
	'&': '&amp;',
	'<': '&lt;',
	'>': '&gt;',
	'"': '&quot;',
	"'": '&#39;'
};

export function escapeHtml(input: string): string {
	return input.replace(/[&<>"']/g, (ch) => HTML_ESCAPES[ch]);
}

const ALLOWED_SCHEMES = ['http:', 'https:', 'mailto:'];

export function safeHref(href: string): string | null {
	if (!href) return null;

	const decoded = href.replace(/&#x([0-9a-f]+);/gi, (_, hex) =>
		String.fromCharCode(parseInt(hex, 16))
	);
	const trimmed = decoded.replace(/[\s\t\n\r]/g, '');

	const colonIdx = trimmed.indexOf(':');
	if (colonIdx === -1) {
		return href;
	}

	const scheme = trimmed.slice(0, colonIdx + 1).toLowerCase();
	if (ALLOWED_SCHEMES.includes(scheme)) {
		return href;
	}
	return null;
}

export function renderHtml(blocks: BlockNode[]): string {
	return blocks.map(renderBlock).join('');
}

function renderBlock(block: BlockNode): string {
	switch (block.kind) {
		case 'paragraph':
			return `<p>${renderInline(block.children)}</p>`;
		case 'heading':
			return `<h${block.level}>${renderInline(block.children)}</h${block.level}>`;
		case 'ulist':
			return `<ul>${block.items.map((c) => `<li>${renderInline(c)}</li>`).join('')}</ul>`;
		case 'olist':
			return `<ol>${block.items.map((c) => `<li>${renderInline(c)}</li>`).join('')}</ol>`;
		case 'blockquote':
			return `<blockquote>${renderInline(block.children)}</blockquote>`;
		case 'codeBlock':
			return `<pre><code>${escapeHtml(block.value)}</code></pre>`;
	}
}

function renderInline(nodes: InlineNode[]): string {
	return nodes.map(renderInlineNode).join('');
}

function renderInlineNode(node: InlineNode): string {
	switch (node.kind) {
		case 'text':
			return escapeHtml(node.value);
		case 'bold':
			return `<strong>${renderInline(node.children)}</strong>`;
		case 'italic':
			return `<em>${renderInline(node.children)}</em>`;
		case 'code':
			return `<code>${escapeHtml(node.value)}</code>`;
		case 'link': {
			const safe = safeHref(node.href);
			const text = renderInline(node.children);
			if (safe === null) {
				return text;
			}
			return `<a href="${escapeHtml(safe)}" rel="noopener noreferrer" target="_blank">${text}</a>`;
		}
	}
}
