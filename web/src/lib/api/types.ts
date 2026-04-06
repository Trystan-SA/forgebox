// Types matching the ForgeBox Go SDK (pkg/sdk/)

export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export type UserRole = 'admin' | 'user';

export interface User {
	id: string;
	name: string;
	email: string;
	role: UserRole;
	team_id?: string;
	created_at: string;
}

export interface Task {
	id: string;
	status: TaskStatus;
	prompt: string;
	result?: string;
	provider: string;
	model: string;
	user_id: string;
	session_id: string;
	cost: number;
	tokens_in: number;
	tokens_out: number;
	error?: string;
	created_at: string;
	started_at?: string;
	completed_at?: string;
}

export interface Session {
	id: string;
	user_id: string;
	provider: string;
	model: string;
	created_at: string;
	updated_at: string;
}

export interface Message {
	role: 'user' | 'assistant' | 'system';
	content?: string;
	tool_calls?: ToolCall[];
	tool_results?: ToolResult[];
}

export interface ToolCall {
	id: string;
	name: string;
	input: string;
}

export interface ToolResult {
	tool_call_id: string;
	content: string;
	is_error: boolean;
}

export interface Provider {
	name: string;
	version: string;
	type: 'provider';
	builtin: boolean;
}

export interface ToolSchema {
	name: string;
	description: string;
	input_schema?: Record<string, unknown>;
}

export interface AuditEntry {
	id: string;
	timestamp: string;
	user_id: string;
	task_id: string;
	action: string;
	tool?: string;
	decision: 'allow' | 'deny';
	reason?: string;
}

export interface CreateTaskRequest {
	prompt: string;
	provider?: string;
	model?: string;
	timeout?: string;
	memory_mb?: number;
	vcpus?: number;
	network_access?: boolean;
}

export type TaskEventType =
	| 'connected'
	| 'status_update'
	| 'text_delta'
	| 'tool_call'
	| 'tool_result'
	| 'error'
	| 'done';

export interface TaskEvent {
	type: TaskEventType;
	text?: string;
	tool_call?: ToolCall;
	result?: ToolResult;
	error?: string;
	status?: TaskStatus;
}

export interface VMPoolStatus {
	pool_size: number;
	active_count: number;
}

export interface Team {
	id: string;
	name: string;
	members: string[];
	created_at: string;
}

export interface Workflow {
	id: string;
	name: string;
	description: string;
	prompt_template: string;
	provider?: string;
	model?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export type AutomationSharing = 'personal' | 'team' | 'org';

export interface Automation {
	id: string;
	name: string;
	description: string;
	created_by: string;
	sharing: AutomationSharing;
	team_id?: string;
	trigger: string;
	nodes: string;
	edges: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateAutomationRequest {
	name: string;
	description?: string;
	sharing?: AutomationSharing;
	team_id?: string;
	trigger?: string;
	nodes?: string;
	edges?: string;
}

export interface UpdateAutomationRequest {
	name?: string;
	description?: string;
	sharing?: AutomationSharing;
	team_id?: string;
	trigger?: string;
	nodes?: string;
	edges?: string;
	enabled?: boolean;
}

export interface TokenUsage {
	user_id: string;
	provider: string;
	model: string;
	tokens_in: number;
	tokens_out: number;
	cost: number;
	period: string;
}

export interface LoginRequest {
	email: string;
	password: string;
}

export interface LoginResponse {
	token: string;
	user: User;
}

export interface SetupStatusResponse {
	setup_required: boolean;
}

export interface SetupRequest {
	name: string;
	email: string;
	password: string;
	setup_password: string;
}

export interface SetupResponse {
	id: string;
	name: string;
	email: string;
	role: string;
}
