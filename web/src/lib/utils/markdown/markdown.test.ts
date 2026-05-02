import { describe, it, expect } from 'vitest';
import { escapeHtml, safeHref } from './render';

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
