import type { BlockNode, InlineNode } from './tokens';
import { parseInline } from './inline';

export function parseBlocks(text: string): BlockNode[] {
	const lines = text.split('\n');
	const blocks: BlockNode[] = [];
	let i = 0;

	while (i < lines.length) {
		const line = lines[i];

		if (line.trim() === '') {
			i++;
			continue;
		}

		if (line.startsWith('```')) {
			const start = i + 1;
			let end = -1;
			for (let j = start; j < lines.length; j++) {
				if (lines[j].startsWith('```')) {
					end = j;
					break;
				}
			}
			if (end === -1) {
				const rest = lines.slice(i).join(' ').trim();
				blocks.push({
					kind: 'paragraph',
					children: [{ kind: 'text', value: rest }]
				});
				break;
			}
			const value = lines.slice(start, end).join('\n');
			blocks.push({ kind: 'codeBlock', value });
			i = end + 1;
			continue;
		}

		const headingMatch = /^(#{1,3}) +(.+)$/.exec(line);
		if (headingMatch) {
			const level = headingMatch[1].length as 1 | 2 | 3;
			blocks.push({
				kind: 'heading',
				level,
				children: parseInline(headingMatch[2])
			});
			i++;
			continue;
		}

		if (/^- +/.test(line)) {
			const items: InlineNode[][] = [];
			while (i < lines.length && /^- +/.test(lines[i])) {
				items.push(parseInline(lines[i].replace(/^- +/, '')));
				i++;
			}
			blocks.push({ kind: 'ulist', items });
			continue;
		}

		if (/^\d+\. +/.test(line)) {
			const items: InlineNode[][] = [];
			while (i < lines.length && /^\d+\. +/.test(lines[i])) {
				items.push(parseInline(lines[i].replace(/^\d+\. +/, '')));
				i++;
			}
			blocks.push({ kind: 'olist', items });
			continue;
		}

		if (/^> ?/.test(line)) {
			const quoted: string[] = [];
			while (i < lines.length && /^> ?/.test(lines[i])) {
				quoted.push(lines[i].replace(/^> ?/, ''));
				i++;
			}
			blocks.push({
				kind: 'blockquote',
				children: parseInline(quoted.join(' '))
			});
			continue;
		}

		const paraLines: string[] = [line];
		i++;
		while (i < lines.length) {
			const next = lines[i];
			if (
				next.trim() === '' ||
				next.startsWith('```') ||
				/^#{1,3} +/.test(next) ||
				/^- +/.test(next) ||
				/^\d+\. +/.test(next) ||
				/^> ?/.test(next)
			) {
				break;
			}
			paraLines.push(next);
			i++;
		}
		blocks.push({
			kind: 'paragraph',
			children: parseInline(paraLines.join(' '))
		});
	}

	return blocks;
}
