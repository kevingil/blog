import * as React from "react";
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
  Wrench,
  Loader2,
  CheckCircle2,
  XCircle,
  ChevronUp,
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
import { cn } from "@/lib/utils";

interface ToolGroupDisplayProps {
  group: ToolGroup;
  onArtifactAction?: (toolId: string, action: 'accept' | 'reject') => void;
}

/**
 * Tools that should use the full card UI (because they have artifacts)
 */
const ARTIFACT_TOOLS = new Set(['edit_text', 'rewrite_document']);

/**
 * Tools that can be expanded to show content (but still use subtle styling)
 */
const EXPANDABLE_TOOLS = new Set(['read_document', 'search_web_sources', 'ask_question']);

/**
 * Check if a tool should use the full card UI
 */
function isArtifactTool(toolName: string): boolean {
  return ARTIFACT_TOOLS.has(toolName);
}

/**
 * Check if a tool can be expanded to show details
 */
function isExpandableTool(toolName: string): boolean {
  return EXPANDABLE_TOOLS.has(toolName);
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
    case 'read_document':
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
 * Get expandable content for a tool (returns null if not expandable or no content)
 */
function getExpandableContent(call: ToolCallRecord): React.ReactNode | null {
  if (call.status !== 'completed' || !call.result) return null;
  
  switch (call.name) {
    case 'read_document': {
      const content = call.result.content as string;
      if (!content) return null;
      return (
        <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono">
          {content}
        </pre>
      );
    }
    case 'search_web_sources':
    case 'get_relevant_sources': {
      const searchResults = (call.result.search_results || call.result.relevant_sources || []) as SearchResult[];
      if (searchResults.length === 0) return null;
      return (
        <div className="flex flex-wrap gap-1.5">
          {searchResults.slice(0, 6).map((sr, idx) => (
            <Source href={sr.url} key={sr.url || idx}>
              <SourceTrigger label={getDomain(sr.url)} showFavicon />
              <SourceContent
                title={sr.title}
                description={sr.summary || sr.text_preview || sr.highlights?.[0]}
                metadata={{ author: sr.author, published: sr.published_date }}
              />
            </Source>
          ))}
        </div>
      );
    }
    case 'ask_question': {
      const answer = call.result.answer as string;
      if (!answer) return null;
      return (
        <div className="text-sm text-muted-foreground">
          {answer}
        </div>
      );
    }
    default:
      return null;
  }
}

/**
 * Subtle inline tool display (for non-artifact tools like read_document, search, etc.)
 */
function SubtleToolDisplay({ call }: { call: ToolCallRecord }) {
  const [isOpen, setIsOpen] = React.useState(false);
  const canExpand = isExpandableTool(call.name) && call.status === 'completed';
  const expandableContent = canExpand ? getExpandableContent(call) : null;
  const hasContent = expandableContent !== null;

  const getStatusIcon = () => {
    switch (call.status) {
      case 'running':
      case 'pending':
        return <Loader2 className="h-3 w-3 text-muted-foreground animate-spin" />;
      case 'completed':
        return <CheckCircle2 className="h-3 w-3 text-green-500" />;
      case 'error':
        return <XCircle className="h-3 w-3 text-red-500" />;
      default:
        return null;
    }
  };

  const handleClick = () => {
    if (hasContent) {
      setIsOpen((prev) => !prev);
    }
  };

  return (
    <div>
      <button
        onClick={handleClick}
        disabled={!hasContent}
        className={cn(
          "flex w-full items-center gap-2 py-1 text-sm text-muted-foreground text-left",
          hasContent && "hover:text-foreground cursor-pointer",
          !hasContent && "cursor-default"
        )}
      >
        <span className="flex-shrink-0 w-3 flex justify-center">
          <span className="text-muted-foreground/70 text-xs">•</span>
        </span>
        <span className="flex-shrink-0 text-muted-foreground/70">
          {getToolIcon(call.name)}
        </span>
        <span className="truncate flex-1">{getToolDisplayName(call.name)}</span>
        {getStatusIcon()}
        {hasContent && (
          <ChevronUp
            className={cn(
              "h-4 w-4 text-muted-foreground/50 transition-transform duration-200 flex-shrink-0",
              !isOpen && "rotate-180"
            )}
          />
        )}
      </button>
      
      {isOpen && hasContent && (
        <div className="ml-5 pl-3 border-l border-border/50 mt-1 max-h-48 overflow-y-auto">
          <div className="py-1.5">
            {expandableContent}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Renders a group of tool calls using appropriate UI based on tool type
 */
export function ToolGroupDisplay({ group }: ToolGroupDisplayProps) {
  return (
    <div className="space-y-1">
      {group.calls.map((call) => {
        // Use full card UI for artifact tools
        if (isArtifactTool(call.name)) {
          return (
            <ToolCall 
              key={call.id} 
              status={mapStatus(call.status)}
              defaultOpen={call.status === 'completed'}
            >
              <ToolCallTrigger icon={getToolIcon(call.name)}>
                {getToolDisplayName(call.name)}
              </ToolCallTrigger>
              <ToolCallContent>
                <ToolResultContent call={call} />
              </ToolCallContent>
            </ToolCall>
          );
        }
        
        // Use subtle inline display for other tools
        return <SubtleToolDisplay key={call.id} call={call} />;
      })}
    </div>
  );
}

export default ToolGroupDisplay;
