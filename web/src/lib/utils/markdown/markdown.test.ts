import { describe, it, expect } from 'vitest';
import { escapeHtml, renderHtml, safeHref } from './render';
import { parseInline } from './inline';

describe('escapeHtml', () => {
	it('escapes the five HTML-significant characters', () => {
		expect(escapeHtml(`<a href="x">&'</a>`)).toBe(
			'&lt;a href=&quot;x&quot;&gt;&amp;&#39;&lt;/a&gt;'
		);
	});

	it('returns input unchanged when no special chars', () => {
		expect(escapeHtml('hello world 123')).toBe('hello world 123');
	});

	it('handles empty string', () => {
		expect(escapeHtml('')).toBe('');
	});
});

describe('safeHref', () => {
	it('allows http URLs', () => {
		expect(safeHref('http://example.com')).toBe('http://example.com');
	});

	it('allows https URLs', () => {
		expect(safeHref('https://example.com/path?q=1')).toBe('https://example.com/path?q=1');
	});

	it('allows mailto URLs', () => {
		expect(safeHref('mailto:a@b.co')).toBe('mailto:a@b.co');
	});

	it('rejects javascript:', () => {
		expect(safeHref('javascript:alert(1)')).toBeNull();
	});

	it('rejects data:', () => {
		expect(safeHref('data:text/html,<script>1</script>')).toBeNull();
	});

	it('rejects vbscript:', () => {
		expect(safeHref('vbscript:msgbox(1)')).toBeNull();
	});

	it('rejects schemes with whitespace tricks', () => {
		expect(safeHref(' javascript:alert(1)')).toBeNull();
		expect(safeHref('java\tscript:alert(1)')).toBeNull();
	});

	it('rejects HTML-entity-encoded javascript scheme', () => {
		expect(safeHref('javascript&#x3A;alert(1)')).toBeNull();
	});

	it('treats bare paths as relative (allowed)', () => {
		expect(safeHref('/foo/bar')).toBe('/foo/bar');
		expect(safeHref('#anchor')).toBe('#anchor');
	});

	it('returns null for empty', () => {
		expect(safeHref('')).toBeNull();
	});
});

function inlineToHtml(text: string): string {
	return renderHtml([{ kind: 'paragraph', children: parseInline(text) }]);
}

describe('parseInline', () => {
	it('plain text', () => {
		expect(inlineToHtml('hello')).toBe('<p>hello</p>');
	});

	it('escapes raw HTML in plain text', () => {
		expect(inlineToHtml('<script>x</script>')).toBe('<p>&lt;script&gt;x&lt;/script&gt;</p>');
	});

	it('bold', () => {
		expect(inlineToHtml('a **b** c')).toBe('<p>a <strong>b</strong> c</p>');
	});

	it('italic', () => {
		expect(inlineToHtml('a *b* c')).toBe('<p>a <em>b</em> c</p>');
	});

	it('inline code', () => {
		expect(inlineToHtml('a `b` c')).toBe('<p>a <code>b</code> c</p>');
	});

	it('escapes HTML inside inline code', () => {
		expect(inlineToHtml('`<b>`')).toBe('<p><code>&lt;b&gt;</code></p>');
	});

	it('link with safe href', () => {
		expect(inlineToHtml('[ok](https://x.co)')).toBe(
			'<p><a href="https://x.co" rel="noopener noreferrer" target="_blank">ok</a></p>'
		);
	});

	it('link with javascript: href renders as escaped text', () => {
		expect(inlineToHtml('[click](javascript:alert(1))')).toBe('<p>click</p>');
	});

	it('link with data: href renders as escaped text', () => {
		expect(inlineToHtml('[click](data:text/html,x)')).toBe('<p>click</p>');
	});

	it('bold containing italic', () => {
		expect(inlineToHtml('**a *b* c**')).toBe('<p><strong>a <em>b</em> c</strong></p>');
	});

	it('** inside backticks renders literal', () => {
		expect(inlineToHtml('`**not bold**`')).toBe('<p><code>**not bold**</code></p>');
	});

	it('unclosed ** renders literal', () => {
		expect(inlineToHtml('**bold')).toBe('<p>**bold</p>');
	});

	it('unclosed * renders literal', () => {
		expect(inlineToHtml('*ital')).toBe('<p>*ital</p>');
	});

	it('unclosed backtick renders literal', () => {
		expect(inlineToHtml('`code')).toBe('<p>`code</p>');
	});

	it('unclosed link renders literal', () => {
		expect(inlineToHtml('[click](no-close')).toBe('<p>[click](no-close</p>');
	});

	it('link text with bold inside', () => {
		expect(inlineToHtml('[**hi**](https://x.co)')).toBe(
			'<p><a href="https://x.co" rel="noopener noreferrer" target="_blank"><strong>hi</strong></a></p>'
		);
	});
});
