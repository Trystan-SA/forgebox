import { parseBlocks } from './block';
import { renderHtml } from './render';

export function renderMarkdown(text: string): string {
	const blocks = parseBlocks(text);
	return renderHtml(blocks);
}
