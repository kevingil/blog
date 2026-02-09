import * as z from 'zod';
import type { 
  ToolGroup, 
  ThinkingBlock, 
  Artifact as NewArtifact,
  TurnStep,
} from "../types";

export const DEFAULT_IMAGE_PROMPT = [
  "A modern, minimalist illustration",
  "A vibrant, colorful scene",
  "A professional business setting",
  "A natural landscape",
  "An abstract design"
];

export const articleSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  content: z.string().min(1, 'Content is required'),
  image_url: z.union([z.string().url(), z.literal('')]).optional(),
  tags: z.array(z.string()),
  // Note: isDraft removed - publish/unpublish is now a separate action
});

export type ArticleFormData = z.infer<typeof articleSchema>;

export type SearchResult = {
  title: string;
  url: string;
  summary?: string;
  author?: string;
  published_date?: string;
  favicon?: string;
  highlights?: string[];
  text_preview?: string;
  has_full_text?: boolean;
};

export type SourceInfo = {
  source_id: string;
  original_title: string;
  original_url: string;
  content_length: number;
  source_type?: string;
  search_query?: string;
};

export type ChatMessage = {
  id?: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  diffState?: 'accepted' | 'rejected';
  diffPreview?: {
    oldText: string;
    newText: string;
    reason?: string;
  };
  meta_data?: {
    artifact?: {
      id: string;
      type: string;
      status: string;
      content: string;
      diff_preview?: string;
      title?: string;
      description?: string;
      applied_at?: string;
    };
    task_status?: any;
    tool_execution?: any;
    tool_group?: ToolGroup;
    thinking?: ThinkingBlock;
    artifacts?: NewArtifact[];
    context?: any;
    user_action?: any;
  };
  // New: Tool group for unified tool call display
  tool_group?: ToolGroup;
  // New: Thinking/chain of thought
  thinking?: ThinkingBlock;
  // Chain of thought steps (reasoning -> tool -> reasoning -> content)
  steps?: TurnStep[];
  // Streaming state
  isReasoningStreaming?: boolean;
  // Legacy tool_context for backward compatibility
  tool_context?: {
    tool_name: string;
    tool_id: string;
    status: 'starting' | 'running' | 'completed' | 'error';
    // Web search specific
    search_query?: string;
    search_results?: SearchResult[];
    sources_created?: SourceInfo[];
    total_found?: number;
    sources_successful?: number;
    // Ask question specific
    answer?: string;
    citations?: Array<{
      url: string;
      title: string;
      author?: string;
      published_date?: string;
    }>;
    // Generic tool fields
    message?: string;
    input?: Record<string, unknown>;
    output?: Record<string, unknown>;
    reason?: string;
  };
  created_at?: string;
};

// Helper function to convert tool names to user-friendly display names
export function getToolDisplayName(toolName: string): string {
  const toolDisplayMap: Record<string, string> = {
    'rewrite_document': 'Rewriting document',
    'edit_text': 'Editing text',
    'analyze_document': 'Analyzing document',
    'generate_image_prompt': 'Generating image prompt',
    'search_web': 'Searching the web',
    'fetch_url': 'Fetching content',
    'search_documents': 'Searching documents',
    'read_file': 'Reading file',
    'write_file': 'Writing file',
    'calculate': 'Calculating',
    'translate': 'Translating',
    'research': 'Researching'
  };
  
  if (toolDisplayMap[toolName]) {
    return toolDisplayMap[toolName];
  }
  
  const friendlyName = toolName
    .replace(/_/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .toLowerCase()
    .replace(/^./, str => str.toUpperCase())
    .trim();
  
  if (toolName.toLowerCase().includes('search')) {
    return `Searching for ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('generate')) {
    return `Generating ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('fetch') || toolName.toLowerCase().includes('get')) {
    return `Fetching ${friendlyName.toLowerCase()}`;
  } else if (toolName.toLowerCase().includes('analyze')) {
    return `Analyzing ${friendlyName.toLowerCase()}`;
  } else {
    return `Using ${friendlyName.toLowerCase()}`;
  }
}
