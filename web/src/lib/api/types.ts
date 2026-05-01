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

export type ProviderType =
	| 'anthropic'
	| 'anthropic-api'
	| 'anthropic-subscription'
	| 'openai'
	| 'ollama';

export interface ProviderModel {
	id: string;
	name: string;
	max_input_tokens: number;
	max_output_tokens: number;
	supports_tools: boolean;
	supports_vision: boolean;
}

export interface Provider {
	name: string;
	version: string;
	type: 'provider';
	builtin: boolean;
	id?: string;
	provider_type?: ProviderType;
	models?: ProviderModel[];
}

export interface CreateProviderRequest {
	type: ProviderType;
	config: Record<string, unknown>;
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

export type AgentSharing = 'personal' | 'team' | 'org';
export type AgentRole = 'worker' | 'orchestrator';

// Tools is stored as a JSON-encoded array on the wire (matches the backend
// AgentRecord schema). Helpers in client.ts marshal to/from string[] so
// dashboard code never sees the JSON encoding.
export interface Agent {
	id: string;
	name: string;
	description: string;
	role: AgentRole;
	system_prompt: string;
	provider: string;
	model: string;
	tools: string;
	sharing: AgentSharing;
	team_id?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface CreateAgentRequest {
	name: string;
	description?: string;
	role?: AgentRole;
	system_prompt?: string;
	provider?: string;
	model?: string;
	tools?: string; // JSON-encoded string[]
	sharing?: AgentSharing;
	team_id?: string;
}

export interface UpdateAgentRequest {
	name?: string;
	description?: string;
	role?: AgentRole;
	system_prompt?: string;
	provider?: string;
	model?: string;
	tools?: string; // JSON-encoded string[]
	sharing?: AgentSharing;
	team_id?: string;
}

export type AppStatus = 'draft' | 'deploying' | 'running' | 'stopped' | 'error';
export type AppSharing = 'personal' | 'team' | 'org';
export type AppTool = 'database' | 'api' | 'ai';

export interface App {
	id: string;
	name: string;
	description: string;
	created_by: string;
	sharing: AppSharing;
	team_id?: string;
	status: AppStatus;
	tools: string;
	config: string;
	url: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateAppRequest {
	name: string;
	description?: string;
	sharing?: AppSharing;
	team_id?: string;
	tools?: string;
	config?: string;
}

export interface UpdateAppRequest {
	name?: string;
	description?: string;
	sharing?: AppSharing;
	team_id?: string;
	status?: AppStatus;
	tools?: string;
	config?: string;
	url?: string;
	enabled?: boolean;
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

// --- Brain ---

export interface Brain {
	id: string;
	automation_id: string;
	embedding_provider?: string;
	embedding_model?: string;
	embedding_dimension: number;
	created_at: string;
	updated_at: string;
}

export interface BrainFile {
	id: string;
	brain_id: string;
	title: string;
	content: string;
	cluster_id?: number;
	created_at: string;
	updated_at: string;
	created_by: string;
}

export interface BrainFileWithMeta extends BrainFile {
	hashtags: string[];
	links: string[];
	score?: number;
}

export interface BrainLink {
	source_file_id: string;
	target_file_id: string;
}

export interface GraphCluster {
	id: number;
	color: string;
	label: string;
}

export interface GraphNode {
	file_id: string;
	title: string;
	x: number;
	y: number;
	cluster_id: number;
	hashtags: string[];
}

export interface BrainGraph {
	brain_id: string;
	clusters: GraphCluster[];
	nodes: GraphNode[];
	links: BrainLink[];
	computed_at: string;
}

export type DreamProposalStatus = 'pending' | 'approved' | 'rejected';

export interface DreamProposal {
	id: string;
	brain_id: string;
	snapshot?: string;
	changes: string;
	summary: string;
	status: DreamProposalStatus;
	created_at: string;
	resolved_at?: string;
	resolved_by?: string;
}

export interface DreamChange {
	action: 'create' | 'edit' | 'delete';
	file_id?: string;
	new_title?: string;
	new_content?: string;
	reason: string;
}
