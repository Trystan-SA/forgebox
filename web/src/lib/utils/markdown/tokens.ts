export type InlineNode =
	| { kind: 'text'; value: string }
	| { kind: 'bold'; children: InlineNode[] }
	| { kind: 'italic'; children: InlineNode[] }
	| { kind: 'code'; value: string }
	| { kind: 'link'; href: string; children: InlineNode[] };

export type BlockNode =
	| { kind: 'paragraph'; children: InlineNode[] }
	| { kind: 'heading'; level: 1 | 2 | 3; children: InlineNode[] }
	| { kind: 'ulist'; items: InlineNode[][] }
	| { kind: 'olist'; items: InlineNode[][] }
	| { kind: 'blockquote'; children: InlineNode[] }
	| { kind: 'codeBlock'; value: string };
