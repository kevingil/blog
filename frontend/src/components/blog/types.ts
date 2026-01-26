// =============================================================================
// Agent Turn Types - New Architecture
// =============================================================================

/**
 * Represents a complete agent turn (thinking -> tool calls -> response)
 */
export interface AgentTurn {
  id: string;
  turn_sequence: number;
  thinking?: ThinkingBlock;
  toolGroup?: ToolGroup;
  content: string;
  artifacts?: Artifact[];
  created_at: string;
  // New: ordered sequence of steps for chain-of-thought display
  steps?: TurnStep[];
  // Streaming state
  isStreaming?: boolean;
  isReasoningStreaming?: boolean;
}

/**
 * Individual step in a turn's chain of thought
 */
export interface TurnStep {
  type: 'reasoning' | 'tool_group' | 'content';
  thinking?: ThinkingBlock;
  toolGroup?: ToolGroup;
  content?: string;
  isStreaming?: boolean;
}

/**
 * Chain of thought reasoning block
 */
export interface ThinkingBlock {
  content: string;
  duration_ms?: number;
  visible?: boolean;
}

/**
 * Group of tool calls that can be executed in parallel
 */
export interface ToolGroup {
  group_id: string;
  status: ToolGroupStatus;
  calls: ToolCallRecord[];
}

export type ToolGroupStatus = 'pending' | 'running' | 'completed' | 'error';

/**
 * Individual tool call record with result
 */
export interface ToolCallRecord {
  id: string;
  name: string;
  input: Record<string, unknown>;
  status: ToolCallStatus;
  result?: Record<string, unknown>;
  error?: string;
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
}

export type ToolCallStatus = 'pending' | 'running' | 'completed' | 'error';

/**
 * Artifact created from tool results
 */
export interface Artifact {
  id: string;
  type: ArtifactType;
  status: ArtifactStatus;
  data: Record<string, unknown>;
}

export type ArtifactType = 'diff' | 'sources' | 'answer' | 'content_generation' | 'image_prompt';
export type ArtifactStatus = 'pending' | 'accepted' | 'rejected';

// =============================================================================
// Stream Event Types
// =============================================================================

export type StreamEventType =
  | 'content_delta'
  | 'reasoning_delta'
  | 'text'
  | 'tool_use'
  | 'tool_result'
  | 'tool_group_start'
  | 'tool_status'
  | 'tool_group_complete'
  | 'full_message'
  | 'artifact'
  | 'user'
  | 'system'
  | 'thinking'
  | 'error'
  | 'done';

/**
 * Stream response from the backend
 */
export interface StreamResponse {
  requestId?: string;
  type: StreamEventType;
  content?: string;
  iteration?: number;

  // Legacy tool fields
  tool_id?: string;
  tool_name?: string;
  tool_input?: Record<string, unknown>;
  tool_result?: {
    content?: string;
    metadata?: string;
    is_error?: boolean;
    is_search?: boolean;
  };

  // New tool group fields
  tool_group?: ToolGroupPayload;
  tool_status?: ToolStatusPayload;

  // Artifact
  artifact?: ArtifactPayload;

  // Thinking
  thinking_content?: string;
  thinking_message?: string; // legacy

  // Full message
  full_message?: FullMessagePayload;

  // Legacy
  role?: string;
  data?: unknown;
  done?: boolean;
  error?: string;
}

export interface ToolGroupPayload {
  group_id: string;
  status: ToolGroupStatus;
  calls: ToolCallPayload[];
}

export interface ToolCallPayload {
  id: string;
  name: string;
  input?: Record<string, unknown>;
  status: ToolCallStatus;
  result?: Record<string, unknown>;
  error?: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
}

export interface ToolStatusPayload {
  group_id: string;
  tool_id: string;
  name: string;
  status: ToolCallStatus;
  result?: Record<string, unknown>;
  error?: string;
  completed_at?: string;
  duration_ms?: number;
}

export interface ArtifactPayload {
  id: string;
  type: ArtifactType;
  status: ArtifactStatus;
  data: Record<string, unknown>;
}

export interface FullMessagePayload {
  id: string;
  article_id: string;
  role: string;
  content: string;
  meta_data: MessageMetaData;
  created_at: string;
}

// =============================================================================
// Message Metadata Types
// =============================================================================

/**
 * Complete metadata structure for a chat message
 */
export interface MessageMetaData {
  // Turn tracking (new)
  turn_id?: string;
  turn_sequence?: number;
  thinking?: ThinkingBlock;
  tool_group?: ToolGroup;
  artifacts?: Artifact[];

  // Legacy fields (for backward compatibility)
  artifact?: LegacyArtifactInfo;
  task_status?: TaskStatus;
  tool_execution?: ToolExecution;
  context?: MessageContext;
  user_action?: UserAction;
}

export interface LegacyArtifactInfo {
  id: string;
  type: string;
  status: string;
  content: string;
  diff_preview?: string;
  title?: string;
  description?: string;
  applied_at?: string;
}

export interface TaskStatus {
  task_id: string;
  name: string;
  status: 'queued' | 'in_progress' | 'completed' | 'failed';
  progress: number;
  started_at: string;
  completed_at?: string;
  error?: string;
}

export interface ToolExecution {
  tool_name: string;
  tool_id: string;
  input: unknown;
  output: unknown;
  error?: string;
  duration_ms: number;
  executed_at: string;
  success: boolean;
}

export interface MessageContext {
  article_id?: string;
  session_id: string;
  request_id?: string;
  document_version?: string;
  document_hash?: string;
  user_id?: string;
}

export interface UserAction {
  action: 'accept' | 'reject' | 'modify';
  timestamp: string;
  artifact_id?: string;
  feedback?: string;
  reason?: string;
}

// =============================================================================
// Tool Categories
// =============================================================================

export type ToolCategory = 'research' | 'analysis' | 'editing' | 'generation';

export const TOOL_CATEGORIES: Record<string, ToolCategory> = {
  search_web_sources: 'research',
  ask_question: 'research',
  get_relevant_sources: 'research',
  fetch_url: 'research',
  analyze_document: 'analysis',
  add_context_from_sources: 'analysis',
  rewrite_document: 'editing',
  edit_text: 'editing',
  generate_text_content: 'generation',
  generate_image_prompt: 'generation',
};

export const PARALLELIZABLE_TOOLS = new Set([
  'search_web_sources',
  'ask_question',
  'get_relevant_sources',
  'fetch_url',
  'analyze_document',
  'add_context_from_sources',
]);

export const ARTIFACT_TOOLS = new Set([
  'search_web_sources',
  'ask_question',
  'get_relevant_sources',
  'rewrite_document',
  'edit_text',
  'generate_text_content',
  'generate_image_prompt',
]);

// =============================================================================
// Tool Display Helpers
// =============================================================================

export const TOOL_DISPLAY_NAMES: Record<string, string> = {
  search_web_sources: 'Web Search',
  ask_question: 'Ask Question',
  get_relevant_sources: 'Find Sources',
  fetch_url: 'Fetch URL',
  analyze_document: 'Analyze Document',
  add_context_from_sources: 'Add Context',
  rewrite_document: 'Rewrite Document',
  edit_text: 'Edit Text',
  generate_text_content: 'Generate Content',
  generate_image_prompt: 'Generate Image Prompt',
};

export function getToolDisplayName(toolName: string): string {
  return TOOL_DISPLAY_NAMES[toolName] || toolName.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

export function getToolCategory(toolName: string): ToolCategory | undefined {
  return TOOL_CATEGORIES[toolName];
}

export function isParallelizable(toolName: string): boolean {
  return PARALLELIZABLE_TOOLS.has(toolName);
}

export function hasArtifact(toolName: string): boolean {
  return ARTIFACT_TOOLS.has(toolName);
}

// =============================================================================
// Search Result Types (for search tools)
// =============================================================================

export interface SearchResult {
  title: string;
  url: string;
  summary?: string;
  author?: string;
  published_date?: string;
  favicon?: string;
  highlights?: string[];
  text_preview?: string;
  has_full_text?: boolean;
}

export interface SourceInfo {
  source_id: string;
  original_title: string;
  original_url: string;
  content_length: number;
  source_type?: string;
  search_query?: string;
}

// =============================================================================
// Answer Types (for ask_question tool)
// =============================================================================

export interface AnswerResult {
  answer: string;
  citations: Citation[];
  cost_info?: Record<string, unknown>;
}

export interface Citation {
  url: string;
  title: string;
  author?: string;
  published_date?: string;
  text?: string;
  favicon?: string;
}

// =============================================================================
// Diff Types (for edit/rewrite tools)
// =============================================================================

export interface DiffData {
  original: string;
  proposed: string;
  reason?: string;
}
