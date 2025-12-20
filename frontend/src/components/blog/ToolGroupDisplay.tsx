import { 
  Search, 
  FileSearch, 
  FileDiff, 
  FileText, 
  ImageIcon, 
  BookOpen, 
  PlusCircle, 
  Link2, 
  MessageSquare,
  Wrench 
} from "lucide-react";
import {
  ToolCall,
  ToolCallTrigger,
  ToolCallContent,
  ToolCallStatusItem,
} from "@/components/prompt-kit/tool-call";
import {
  Source,
  SourceContent,
  SourceTrigger,
} from "@/components/prompt-kit/source";
import type { 
  ToolGroup, 
  ToolCallRecord, 
  ToolCallStatus,
  SearchResult,
  AnswerResult,
} from "./types";
import { getToolDisplayName } from "./types";

interface ToolGroupDisplayProps {
  group: ToolGroup;
  onArtifactAction?: (toolId: string, action: 'accept' | 'reject') => void;
}

/**
 * Maps internal tool status to ToolCall component status
 */
function mapStatus(status: ToolCallStatus): 'pending' | 'running' | 'completed' | 'error' {
  switch (status) {
    case 'pending':
      return 'pending';
    case 'running':
      return 'running';
    case 'completed':
      return 'completed';
    case 'error':
      return 'error';
    default:
      return 'pending';
  }
}

/**
 * Gets the appropriate icon for a tool
 */
function getToolIcon(toolName: string) {
  switch (toolName) {
    case 'search_web_sources':
      return <Search className="h-4 w-4" />;
    case 'ask_question':
      return <MessageSquare className="h-4 w-4" />;
    case 'get_relevant_sources':
      return <FileSearch className="h-4 w-4" />;
    case 'fetch_url':
      return <Link2 className="h-4 w-4" />;
    case 'rewrite_document':
    case 'edit_text':
      return <FileDiff className="h-4 w-4" />;
    case 'analyze_document':
      return <BookOpen className="h-4 w-4" />;
    case 'add_context_from_sources':
      return <PlusCircle className="h-4 w-4" />;
    case 'generate_text_content':
      return <FileText className="h-4 w-4" />;
    case 'generate_image_prompt':
      return <ImageIcon className="h-4 w-4" />;
    default:
      return <Wrench className="h-4 w-4" />;
  }
}

/**
 * Extract domain from URL for display
 */
function getDomain(url: string) {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname.replace('www.', '');
  } catch {
    return url;
  }
}

/**
 * Renders the content for a specific tool call result
 */
function ToolResultContent({ call }: { call: ToolCallRecord }) {
  const { name, status, result, error } = call;

  // Show error state
  if (status === 'error') {
    return (
      <ToolCallStatusItem status="error">
        {error || 'Tool execution failed'}
      </ToolCallStatusItem>
    );
  }

  // Show running state
  if (status === 'running' || status === 'pending') {
    return (
      <ToolCallStatusItem status="running">
        Processing...
      </ToolCallStatusItem>
    );
  }

  // Handle different tool types
  switch (name) {
    case 'search_web_sources':
    case 'get_relevant_sources':
      return <SearchResultsContent result={result} />;
    
    case 'ask_question':
      return <AnswerResultContent result={result} />;
    
    case 'rewrite_document':
    case 'edit_text':
      return <DiffResultContent result={result} />;
    
    case 'analyze_document':
      return <AnalysisResultContent result={result} />;
    
    case 'generate_text_content':
      return <GeneratedContentResult result={result} />;
    
    case 'generate_image_prompt':
      return <ImagePromptResult result={result} />;
    
    default:
      return (
        <ToolCallStatusItem status="completed">
          Completed
        </ToolCallStatusItem>
      );
  }
}

/**
 * Renders search results with source pills
 */
function SearchResultsContent({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const searchResults = (result.search_results || result.relevant_sources || []) as SearchResult[];
  const totalFound = (result.total_found as number) || searchResults.length;
  const sourcesCreated = (result.sources_successful as number) || 0;

  return (
    <div className="space-y-2">
      <ToolCallStatusItem status="completed">
        Found {totalFound} {totalFound === 1 ? 'result' : 'results'}
        {sourcesCreated > 0 && `, created ${sourcesCreated} sources`}
      </ToolCallStatusItem>
      
      {searchResults.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {searchResults.slice(0, 6).map((sr, idx) => (
            <Source href={sr.url} key={sr.url || idx}>
              <SourceTrigger 
                label={getDomain(sr.url)} 
                showFavicon 
              />
              <SourceContent
                title={sr.title}
                description={sr.summary || sr.text_preview || sr.highlights?.[0]}
                metadata={{
                  author: sr.author,
                  published: sr.published_date
                }}
              />
            </Source>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Renders answer result with citations
 */
function AnswerResultContent({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const answer = result.answer as string;
  const citations = (result.citations || []) as Array<{
    url: string;
    title: string;
    author?: string;
    published_date?: string;
    favicon?: string;
  }>;

  return (
    <div className="space-y-2">
      <ToolCallStatusItem status="completed">
        Answer found with {citations.length} {citations.length === 1 ? 'citation' : 'citations'}
      </ToolCallStatusItem>
      
      {answer && (
        <div className="text-sm text-muted-foreground bg-muted/50 rounded-md p-2">
          {answer}
        </div>
      )}
      
      {citations.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {citations.slice(0, 4).map((citation, idx) => (
            <Source href={citation.url} key={citation.url || idx}>
              <SourceTrigger 
                label={getDomain(citation.url)} 
                showFavicon 
              />
              <SourceContent
                title={citation.title}
                metadata={{
                  author: citation.author,
                  published: citation.published_date
                }}
              />
            </Source>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Renders diff result summary
 */
function DiffResultContent({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const reason = result.reason as string;
  const patch = result.patch as { summary?: { additions?: number; deletions?: number } };

  return (
    <div className="space-y-2">
      <ToolCallStatusItem status="completed">
        Changes prepared
        {patch?.summary && (
          <span className="text-xs ml-2">
            (+{patch.summary.additions || 0} / -{patch.summary.deletions || 0})
          </span>
        )}
      </ToolCallStatusItem>
      
      {reason && (
        <div className="text-xs text-muted-foreground">
          {reason}
        </div>
      )}
    </div>
  );
}

/**
 * Renders analysis result
 */
function AnalysisResultContent({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const focusArea = result.focus_area as string;

  return (
    <ToolCallStatusItem status="completed">
      Analysis complete{focusArea && ` (${focusArea})`}
    </ToolCallStatusItem>
  );
}

/**
 * Renders generated content result
 */
function GeneratedContentResult({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const sourcesIncluded = result.sources_included as number;

  return (
    <ToolCallStatusItem status="completed">
      Content generated
      {sourcesIncluded > 0 && ` using ${sourcesIncluded} sources`}
    </ToolCallStatusItem>
  );
}

/**
 * Renders image prompt result
 */
function ImagePromptResult({ result }: { result?: Record<string, unknown> }) {
  if (!result) return null;

  const prompt = result.prompt as string;

  return (
    <div className="space-y-2">
      <ToolCallStatusItem status="completed">
        Image prompt generated
      </ToolCallStatusItem>
      
      {prompt && (
        <div className="text-xs text-muted-foreground bg-muted/50 rounded-md p-2 line-clamp-3">
          {prompt}
        </div>
      )}
    </div>
  );
}

/**
 * Renders a group of tool calls using Prompt Kit ToolCall components
 */
export function ToolGroupDisplay({ group }: ToolGroupDisplayProps) {
  return (
    <div className="space-y-2">
      {group.calls.map((call) => (
        <ToolCall 
          key={call.id} 
          status={mapStatus(call.status)}
          defaultOpen={call.status === 'completed'}
        >
          <ToolCallTrigger icon={getToolIcon(call.name)}>
            {getToolDisplayName(call.name)}
            {call.status === 'completed' && call.duration_ms != null && call.duration_ms > 0 && (
              <span className="text-xs text-muted-foreground/70 ml-2">
                {Math.round(call.duration_ms)}ms
              </span>
            )}
          </ToolCallTrigger>
          <ToolCallContent>
            <ToolResultContent call={call} />
          </ToolCallContent>
        </ToolCall>
      ))}
    </div>
  );
}

export default ToolGroupDisplay;
