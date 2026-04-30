import { isTauri } from '$lib/platform';

// Wire envelope shared by every server message.
type SocketMessage = {
	type: string;
	payload?: unknown;
};

type Handler = (payload: unknown) => void;

type State = {
	connected: boolean;
};

export const socketStore = $state<State>({ connected: false });

const handlers = new Map<string, Set<Handler>>();
let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelayMs = 1_000;
const maxReconnectDelayMs = 30_000;

function getWSUrl(): string {
	if (isTauri()) {
		const stored = localStorage.getItem('forgebox_api_url');
		const base = stored || 'http://localhost:8420/api/v1';
		return base.replace(/^http/, 'ws') + '/ws';
	}
	const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${proto}//${window.location.host}/api/v1/ws`;
}

function getToken(): string | null {
	return localStorage.getItem('forgebox_token');
}

export function subscribe(type: string, handler: Handler): () => void {
	let set = handlers.get(type);
	if (!set) {
		set = new Set();
		handlers.set(type, set);
	}
	set.add(handler);
	return () => {
		set?.delete(handler);
	};
}

function dispatch(msg: SocketMessage) {
	const set = handlers.get(msg.type);
	if (!set) return;
	for (const h of set) {
		try {
			h(msg.payload);
		} catch (err) {
			console.error('socket handler threw', err);
		}
	}
}

function send(msg: SocketMessage) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	socket.send(JSON.stringify(msg));
}

function scheduleReconnect() {
	if (reconnectTimer) return;
	reconnectTimer = setTimeout(() => {
		reconnectTimer = null;
		connect();
	}, reconnectDelayMs);
	reconnectDelayMs = Math.min(reconnectDelayMs * 2, maxReconnectDelayMs);
}

function connect() {
	if (socket && socket.readyState <= WebSocket.OPEN) return;

	const ws = new WebSocket(getWSUrl());
	socket = ws;

	ws.onopen = () => {
		const token = getToken();
		// Without a token, the server will close the connection. Wait for a
		// real login before retrying so we don't hammer the server.
		if (!token) {
			ws.close();
			return;
		}
		send({ type: 'auth', payload: { token } });
	};

	ws.onmessage = (e) => {
		let msg: SocketMessage;
		try {
			msg = JSON.parse(e.data);
		} catch {
			return;
		}

		switch (msg.type) {
			case 'auth_ok':
				socketStore.connected = true;
				reconnectDelayMs = 1_000;
				return;
			case 'auth_error':
				// Bad token — close and let the auth flow re-trigger a connect.
				ws.close();
				return;
			case 'ping':
				send({ type: 'pong' });
				return;
			default:
				dispatch(msg);
		}
	};

	ws.onclose = () => {
		socketStore.connected = false;
		socket = null;
		scheduleReconnect();
	};

	ws.onerror = () => {
		// onclose will fire next; reconnect logic lives there.
	};
}

// Call after login so a freshly stored token gets picked up immediately
// instead of waiting for the next reconnect tick.
export function reconnect() {
	if (reconnectTimer) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
	if (socket) {
		try {
			socket.close();
		} catch {
			// ignore
		}
		socket = null;
	}
	reconnectDelayMs = 1_000;
	connect();
}

if (typeof window !== 'undefined') {
	connect();
}
