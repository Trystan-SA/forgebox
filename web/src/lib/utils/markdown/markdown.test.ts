import { describe, it, expect } from 'vitest';
import { escapeHtml, renderHtml, safeHref } from './render';
import { parseInline } from './inline';
import { parseBlocks } from './block';
import { renderMarkdown } from './index';

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

function blocksToHtml(text: string): string {
	return renderHtml(parseBlocks(text));
}

describe('parseBlocks', () => {
	it('single paragraph', () => {
		expect(blocksToHtml('hello world')).toBe('<p>hello world</p>');
	});

	it('two paragraphs separated by blank line', () => {
		expect(blocksToHtml('one\n\ntwo')).toBe('<p>one</p><p>two</p>');
	});

	it('soft-wrapped paragraph keeps newlines as spaces', () => {
		expect(blocksToHtml('one\ntwo')).toBe('<p>one two</p>');
	});

	it('h1', () => {
		expect(blocksToHtml('# Title')).toBe('<h1>Title</h1>');
	});

	it('h2', () => {
		expect(blocksToHtml('## Title')).toBe('<h2>Title</h2>');
	});

	it('h3', () => {
		expect(blocksToHtml('### Title')).toBe('<h3>Title</h3>');
	});

	it('headings with inline formatting', () => {
		expect(blocksToHtml('## hello **world**')).toBe('<h2>hello <strong>world</strong></h2>');
	});

	it('unordered list', () => {
		expect(blocksToHtml('- a\n- b')).toBe('<ul><li>a</li><li>b</li></ul>');
	});

	it('ordered list', () => {
		expect(blocksToHtml('1. a\n2. b')).toBe('<ol><li>a</li><li>b</li></ol>');
	});

	it('blockquote', () => {
		expect(blocksToHtml('> quoted')).toBe('<blockquote>quoted</blockquote>');
	});

	it('fenced code block preserves contents and escapes them', () => {
		expect(blocksToHtml('```\n<b>x</b>\n```')).toBe(
			'<pre><code>&lt;b&gt;x&lt;/b&gt;</code></pre>'
		);
	});

	it('fenced code block with language tag (tag ignored)', () => {
		expect(blocksToHtml('```js\nconst x = 1;\n```')).toBe(
			'<pre><code>const x = 1;</code></pre>'
		);
	});

	it('fenced code block without closer renders literal text', () => {
		expect(blocksToHtml('```\nunclosed')).toBe('<p>``` unclosed</p>');
	});

	it('heading then list', () => {
		expect(blocksToHtml('# Title\n\n- a\n- b')).toBe(
			'<h1>Title</h1><ul><li>a</li><li>b</li></ul>'
		);
	});

	it('list then paragraph', () => {
		expect(blocksToHtml('- a\n- b\n\nthen prose')).toBe(
			'<ul><li>a</li><li>b</li></ul><p>then prose</p>'
		);
	});

	it('paragraph then code block', () => {
		expect(blocksToHtml('intro\n\n```\ncode\n```')).toBe(
			'<p>intro</p><pre><code>code</code></pre>'
		);
	});

	it('empty input', () => {
		expect(blocksToHtml('')).toBe('');
	});

	it('whitespace-only input', () => {
		expect(blocksToHtml('   \n\n  ')).toBe('');
	});
});

describe('renderMarkdown end-to-end', () => {
	it('mixed prose, list, code', () => {
		const input = `# Hello

This is **bold** and *italic*.

- one
- two

\`\`\`
code
\`\`\``;
		const expected =
			'<h1>Hello</h1>' +
			'<p>This is <strong>bold</strong> and <em>italic</em>.</p>' +
			'<ul><li>one</li><li>two</li></ul>' +
			'<pre><code>code</code></pre>';
		expect(renderMarkdown(input)).toBe(expected);
	});
});

describe('renderMarkdown XSS table', () => {
	const cases: Array<{ name: string; input: string; expected: string }> = [
		{
			name: 'raw script tag escaped',
			input: '<script>alert(1)</script>',
			expected: '<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>'
		},
		{
			name: 'javascript: href stripped',
			input: '[click](javascript:alert(1))',
			expected: '<p>click</p>'
		},
		{
			name: 'data: href stripped',
			input: '[click](data:text/html,<script>1</script>)',
			expected: '<p>click</p>'
		},
		{
			name: 'entity-encoded javascript scheme stripped',
			input: '[click](javascript&#x3A;alert(1))',
			expected: '<p>click</p>'
		},
		{
			name: 'tags inside bold are escaped',
			input: '**<img src=x onerror=alert(1)>**',
			expected: '<p><strong>&lt;img src=x onerror=alert(1)&gt;</strong></p>'
		},
		{
			name: 'attribute injection in href is escaped',
			input: '[x](https://a.co" onmouseover="alert(1))',
			expected:
				'<p><a href="https://a.co&quot; onmouseover=&quot;alert(1)" rel="noopener noreferrer" target="_blank">x</a></p>'
		},
		{
			name: 'tags inside code are escaped',
			input: '`<script>x</script>`',
			expected: '<p><code>&lt;script&gt;x&lt;/script&gt;</code></p>'
		}
	];

	for (const c of cases) {
		it(c.name, () => {
			expect(renderMarkdown(c.input)).toBe(c.expected);
		});
	}
});

describe('streaming partials', () => {
	it('partial bold marker renders literal', () => {
		expect(renderMarkdown('**bo')).toBe('<p>**bo</p>');
	});

	it('partial bold with closer renders strong', () => {
		expect(renderMarkdown('**bold**')).toBe('<p><strong>bold</strong></p>');
	});

	it('partial code-fence renders as paragraph', () => {
		expect(renderMarkdown('```\nstreaming')).toBe('<p>``` streaming</p>');
	});
});
