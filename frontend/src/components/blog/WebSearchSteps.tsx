import { Search } from "lucide-react"
import {
  Source,
  SourceContent,
  SourceTrigger,
} from "@/components/prompt-kit/source"
import {
  ToolCall,
  ToolCallTrigger,
  ToolCallContent,
  ToolCallStatusItem,
} from "@/components/prompt-kit/tool-call"

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

export interface WebSearchToolContext {
  tool_name: string;
  tool_id: string;
  status: 'starting' | 'running' | 'completed' | 'error';
  search_query?: string;
  search_results?: SearchResult[];
  sources_created?: SourceInfo[];
  total_found?: number;
  sources_successful?: number;
  message?: string;
}

interface WebSearchStepsProps {
  tool_context: WebSearchToolContext;
}

// Extract domain from URL for display
function getDomain(url: string) {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname.replace('www.', '');
  } catch {
    return url;
  }
}

// Convert tool context status to ToolCall status
function mapStatus(status: WebSearchToolContext['status']): 'pending' | 'running' | 'completed' | 'error' {
  switch (status) {
    case 'starting':
      return 'pending'
    case 'running':
      return 'running'
    case 'completed':
      return 'completed'
    case 'error':
      return 'error'
    default:
      return 'pending'
  }
}

/**
 * Content component for web search tool - renders inside ToolCall
 */
export function WebSearchContent({ tool_context }: WebSearchStepsProps) {
  const { status, search_results, sources_created, total_found, sources_successful, message } = tool_context;

  return (
    <div className="space-y-2">
      {/* Step 1: Searching */}
      <ToolCallStatusItem status={status === 'starting' || status === 'running' ? 'running' : 'completed'}>
        Searching across curated sources...
      </ToolCallStatusItem>
      
      {/* Step 2: Results found */}
      {total_found !== undefined && total_found > 0 && (
        <>
          <ToolCallStatusItem status="completed">
            Found {total_found} {total_found === 1 ? 'match' : 'matches'}
          </ToolCallStatusItem>
          
          {/* Display sources */}
          {search_results && search_results.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {search_results.slice(0, 6).map((result, idx) => (
                <Source href={result.url} key={result.url || idx}>
                  <SourceTrigger 
                    label={getDomain(result.url)} 
                    showFavicon 
                  />
                  <SourceContent
                    title={result.title}
                    description={result.summary || result.text_preview || result.highlights?.[0]}
                    metadata={{
                      author: result.author,
                      published: result.published_date
                    }}
                  />
                </Source>
              ))}
            </div>
          )}
        </>
      )}
      
      {/* Step 3: Creating sources */}
      {sources_created && sources_created.length > 0 && (
        <ToolCallStatusItem status="completed">
          Created {sources_successful || sources_created.length} reference {sources_successful === 1 ? 'source' : 'sources'}
        </ToolCallStatusItem>
      )}

      {/* Error state */}
      {status === 'error' && (
        <ToolCallStatusItem status="error">
          {message || 'Search failed'}
        </ToolCallStatusItem>
      )}

      {/* No results state */}
      {status === 'completed' && total_found === 0 && (
        <ToolCallStatusItem status="completed">
          No results found for this query
        </ToolCallStatusItem>
      )}
    </div>
  );
}

/**
 * Full web search tool component with ToolCall wrapper
 * This is the main export for use in the chat messages
 */
export function WebSearchSteps({ tool_context }: WebSearchStepsProps) {
  const { status, search_query } = tool_context;

  return (
    <ToolCall 
      status={mapStatus(status)} 
      defaultOpen={status === 'completed'}
    >
      <ToolCallTrigger icon={<Search className="h-4 w-4" />}>
        Web search: {search_query || 'researching...'}
      </ToolCallTrigger>
      
      <ToolCallContent>
        <WebSearchContent tool_context={tool_context} />
      </ToolCallContent>
    </ToolCall>
  );
}
