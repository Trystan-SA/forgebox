// YAML syntax highlighter for read-only previews in the dashboard.
//
// This is NOT a YAML parser — it does not validate structure, handle flow
// style, anchors, multi-document streams, or block scalars robustly. It is a
// deliberately small line-oriented tokenizer whose only job is to colorize the
// YAML served by the backend (e.g. GET /automations/{id}/yaml) so a reader can
// scan the document visually.
//
// Used by: `YamlPreviewModal.svelte` and any future "preview as YAML" modals
// for apps, agents, or other backend-owned resources. The backend is always
// the source of truth for the YAML format itself — this file only colors the
// output it returns.
export type YamlTokKind =
	| 'indent'
	| 'marker'
	| 'key'
	| 'punc'
	| 'string'
	| 'number'
	| 'bool'
	| 'null'
	| 'comment'
	| 'plain';

export interface YamlTok {
	text: string;
	kind: YamlTokKind;
}

export function highlightYaml(line: string): YamlTok[] {
	const tokens: YamlTok[] = [];
	const indentMatch = /^[ \t]+/.exec(line);
	if (indentMatch) tokens.push({ text: indentMatch[0], kind: 'indent' });
	let rest = indentMatch ? line.slice(indentMatch[0].length) : line;

	if (rest.startsWith('- ')) {
		tokens.push({ text: '- ', kind: 'marker' });
		rest = rest.slice(2);
	} else if (rest === '-') {
		tokens.push({ text: '-', kind: 'marker' });
		return tokens;
	}

	let commentText = '';
	const hashIdx = findUnquotedHash(rest);
	if (hashIdx >= 0) {
		commentText = rest.slice(hashIdx);
		rest = rest.slice(0, hashIdx);
	}

	const keyMatch = /^([A-Za-z_][\w-]*|"(?:\\.|[^"\\])*")\s*:/.exec(rest);
	if (keyMatch) {
		tokens.push({ text: keyMatch[1], kind: 'key' });
		tokens.push({ text: ':', kind: 'punc' });
		let after = rest.slice(keyMatch[0].length);
		const spaceMatch = /^\s+/.exec(after);
		if (spaceMatch) {
			tokens.push({ text: spaceMatch[0], kind: 'plain' });
			after = after.slice(spaceMatch[0].length);
		}
		if (after) tokens.push(classifyValue(after));
	} else if (rest.length > 0) {
		tokens.push(classifyValue(rest));
	}

	if (commentText) tokens.push({ text: commentText, kind: 'comment' });
	return tokens;
}

export function tokenizeYaml(text: string): YamlTok[][] {
	if (!text) return [];
	return text.split('\n').map(highlightYaml);
}

function findUnquotedHash(s: string): number {
	let inStr = false;
	for (let i = 0; i < s.length; i++) {
		const c = s[i];
		if (c === '"' && s[i - 1] !== '\\') inStr = !inStr;
		if (c === '#' && !inStr && (i === 0 || /\s/.test(s[i - 1]))) return i;
	}
	return -1;
}

function classifyValue(s: string): YamlTok {
	if (/^"(\\.|[^"\\])*"$/.test(s)) return { text: s, kind: 'string' };
	if (/^-?\d+(\.\d+)?$/.test(s)) return { text: s, kind: 'number' };
	if (/^(true|false)$/i.test(s)) return { text: s, kind: 'bool' };
	if (/^(null|~)$/i.test(s)) return { text: s, kind: 'null' };
	return { text: s, kind: 'plain' };
}
