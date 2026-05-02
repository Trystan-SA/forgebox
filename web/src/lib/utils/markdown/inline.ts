import type { InlineNode } from './tokens';

export function parseInline(text: string): InlineNode[] {
	const out: InlineNode[] = [];
	let i = 0;
	let buf = '';

	function flush() {
		if (buf.length > 0) {
			out.push({ kind: 'text', value: buf });
			buf = '';
		}
	}

	while (i < text.length) {
		const ch = text[i];

		if (ch === '`') {
			const end = text.indexOf('`', i + 1);
			if (end === -1) {
				buf += ch;
				i++;
				continue;
			}
			flush();
			out.push({ kind: 'code', value: text.slice(i + 1, end) });
			i = end + 1;
			continue;
		}

		if (ch === '*' && text[i + 1] === '*') {
			const end = text.indexOf('**', i + 2);
			if (end === -1) {
				buf += '**';
				i += 2;
				continue;
			}
			flush();
			out.push({ kind: 'bold', children: parseInline(text.slice(i + 2, end)) });
			i = end + 2;
			continue;
		}

		if (ch === '*') {
			const end = text.indexOf('*', i + 1);
			if (end === -1) {
				buf += ch;
				i++;
				continue;
			}
			flush();
			out.push({ kind: 'italic', children: parseInline(text.slice(i + 1, end)) });
			i = end + 1;
			continue;
		}

		if (ch === '[') {
			const closeText = text.indexOf(']', i + 1);
			if (closeText === -1 || text[closeText + 1] !== '(') {
				buf += ch;
				i++;
				continue;
			}
			let depth = 1;
			let closeHref = -1;
			for (let j = closeText + 2; j < text.length; j++) {
				if (text[j] === '(') depth++;
				else if (text[j] === ')') {
					depth--;
					if (depth === 0) {
						closeHref = j;
						break;
					}
				}
			}
			if (closeHref === -1) {
				buf += ch;
				i++;
				continue;
			}
			flush();
			const linkText = text.slice(i + 1, closeText);
			const href = text.slice(closeText + 2, closeHref);
			out.push({ kind: 'link', href, children: parseInline(linkText) });
			i = closeHref + 1;
			continue;
		}

		buf += ch;
		i++;
	}

	flush();
	return out;
}
