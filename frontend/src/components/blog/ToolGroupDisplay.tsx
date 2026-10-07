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
  Check,
  X,
  ChevronDown,
  Lightbulb,
} from "lucide-react";
import {
  Source,
  SourceContent,
  SourceTrigger,
} from "@/components/prompt-kit/source";
import { DiffArtifact } from "./DiffArtifact";
import type { 
  ToolGroup, 
  ToolCallRecord, 
  SearchResult,
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
const ARTIFACT_TOOLS = new Set(['replace_lines', 'rewrite_document', 'apply_patch']);

/**
 * Check if a tool should use the full card UI
 */
function isArtifactTool(toolName: string): boolean {
  return ARTIFACT_TOOLS.has(toolName);
}

/**
 * Gets the appropriate icon for a tool
 */
function getToolIcon(toolName: string) {
  switch (toolName) {
    case 'list_insights':
    case 'search_insights':
      return <Lightbulb className="h-3.5 w-3.5" />;
    case 'search_web_sources':
    case 'deep_research':
    case 'web_search':
      return <Search className="h-3.5 w-3.5" />;
    case 'ask_question':
      return <MessageSquare className="h-3.5 w-3.5" />;
    case 'get_relevant_sources':
      return <FileSearch className="h-3.5 w-3.5" />;
    case 'fetch_url':
      return <Link2 className="h-3.5 w-3.5" />;
    case 'rewrite_document':
    case 'replace_lines':
    case 'apply_patch':
      return <FileDiff className="h-3.5 w-3.5" />;
    case 'analyze_document':
    case 'read_document':
      return <BookOpen className="h-3.5 w-3.5" />;
    case 'add_context_from_sources':
      return <PlusCircle className="h-3.5 w-3.5" />;
    case 'generate_text_content':
      return <FileText className="h-3.5 w-3.5" />;
    case 'generate_image_prompt':
      return <ImageIcon className="h-3.5 w-3.5" />;
    default:
      return <Wrench className="h-3.5 w-3.5" />;
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
 * Get expandable content for a tool (returns null if not expandable or no content)
 */
function getExpandableContent(call: ToolCallRecord): React.ReactNode | null {
  if (call.status !== 'completed' || !call.result) return null;
  
  switch (call.name) {
    case 'list_insights':
    case 'search_insights':
      return <InsightResults result={call.result} />;
    case 'set_title': {
      const title = stringField(call.result, 'new_title') || stringField(call.input, 'title');
      return title ? <p className="text-foreground/85">{title}</p> : null;
    }
    case 'update_sources': {
      const sources = Array.isArray(call.result.sources) ? call.result.sources as Array<Record<string, unknown>> : [];
      if (sources.length === 0) return <p>No sources on this article.</p>;
      return (
        <ul className="space-y-1">
          {sources.map((source, index) => (
            <li key={String(source.id || index)} className="truncate">
              <span className="text-foreground/85">{String(source.title || source.url || 'Source')}</span>
              {typeof source.url === 'string' && source.url ? (
                <span className="ml-1 text-muted-foreground">{source.url}</span>
              ) : null}
            </li>
          ))}
        </ul>
      );
    }
    case 'read_document': {
      const content = call.result.content as string;
      const title = stringField(call.result, 'title');
      if (!content && !title) return null;
      return (
        <div className="space-y-1">
          {title ? <p className="text-foreground/85">{title}</p> : null}
          {content ? (
            <pre className="whitespace-pre-wrap font-mono text-xs text-muted-foreground">
              {content}
            </pre>
          ) : null}
        </div>
      );
    }
    case 'search_web_sources':
    case 'deep_research':
    case 'web_search':
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
        <div className="text-xs text-muted-foreground/70 dark:text-muted-foreground/60">
          {answer}
        </div>
      );
    }
    case 'replace_lines': {
      const oldStr = (call.result.old_str || call.result.original_text || '') as string;
      const newStr = (call.result.new_str || call.result.new_text || call.result.new_markdown || '') as string;
      const reason = call.result.reason as string;
      if (!oldStr && !newStr) return null;
      return (
        <div className="space-y-1.5">
          {reason && (
            <div className="text-xs text-muted-foreground/70 dark:text-muted-foreground/60">{reason}</div>
          )}
          {oldStr && (
            <pre className="text-xs font-mono whitespace-pre-wrap text-red-400/80 dark:text-red-400/60 line-clamp-4">
              {oldStr}
            </pre>
          )}
          {newStr && (
            <pre className="text-xs font-mono whitespace-pre-wrap text-green-500/80 dark:text-green-400/60 line-clamp-4">
              {newStr}
            </pre>
          )}
        </div>
      );
    }
    case 'sandbox': {
      const stdout = (call.result.stdout || '') as string;
      const stderr = (call.result.stderr || '') as string;
      const output = stdout || stderr;
      if (!output) return null;
      return (
        <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono">
          {output}
        </pre>
      );
    }
    default:
      return <GenericToolDetail call={call} />;
  }
}

function stringField(source: Record<string, unknown> | undefined, key: string): string {
  const value = source?.[key];
  return typeof value === 'string' ? value.trim() : '';
}

type InsightRow = {
  id?: string;
  title?: string;
  summary?: string;
  topic?: string;
  content?: string;
  key_points?: string[];
};

function InsightResults({ result }: { result: Record<string, unknown> }) {
  const insights = (Array.isArray(result.insights) ? result.insights : []) as InsightRow[];
  if (insights.length === 0) {
    return <p>No insights yet.</p>;
  }
  return (
    <ul className="space-y-2">
      {insights.map((insight, index) => (
        <li key={insight.id || index}>
          <p className="text-foreground/90">{insight.title || 'Untitled insight'}</p>
          {insight.topic ? (
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{insight.topic}</p>
          ) : null}
          {insight.summary ? <p className="text-muted-foreground">{insight.summary}</p> : null}
          {insight.key_points && insight.key_points.length > 0 ? (
            <ul className="mt-1 list-disc pl-4 text-muted-foreground">
              {insight.key_points.slice(0, 4).map((point) => (
                <li key={point}>{point}</li>
              ))}
            </ul>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

function GenericToolDetail({ call }: { call: ToolCallRecord }) {
  const result = call.result || {};
  const titled = Object.values(result).find((value) => (
    Array.isArray(value) && value.some((item) => item && typeof item === 'object' && 'title' in (item as object))
  )) as Array<Record<string, unknown>> | undefined;
  if (titled && titled.length > 0) {
    return (
      <ul className="space-y-1">
        {titled.slice(0, 8).map((item, index) => (
          <li key={String(item.id || item.url || index)} className="truncate text-foreground/85">
            {String(item.title || item.url || item.name || 'Result')}
          </li>
        ))}
      </ul>
    );
  }
  const lines = Object.entries(result)
    .filter(([key, value]) => !['tool_name', 'is_error'].includes(key) && (typeof value === 'string' || typeof value === 'number'))
    .slice(0, 6);
  if (lines.length === 0) {
    const reason = stringField(call.input, 'reason') || stringField(call.input, 'query');
    return reason ? <p>{reason}</p> : null;
  }
  return (
    <dl className="space-y-1">
      {lines.map(([key, value]) => (
        <div key={key}>
          <dt className="text-[11px] uppercase tracking-wide">{key.replace(/_/g, ' ')}</dt>
          <dd className="whitespace-pre-wrap text-foreground/85">{String(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

const TOOL_ACTIVITY: Record<string, { done: string; running: string }> = {
  list_insights: { done: 'Read insights', running: 'Reading insights' },
  search_insights: { done: 'Searched insights', running: 'Searching insights' },
  read_document: { done: 'Read article', running: 'Reading article' },
  set_title: { done: 'Set title', running: 'Setting title' },
  update_sources: { done: 'Updated sources', running: 'Updating sources' },
  web_search: { done: 'Searched the web', running: 'Searching the web' },
  search_web_sources: { done: 'Searched the web', running: 'Searching the web' },
  deep_research: { done: 'Researched', running: 'Researching' },
  get_relevant_sources: { done: 'Found sources', running: 'Finding sources' },
  select_sources_for_edit: { done: 'Selected sources', running: 'Selecting sources' },
  apply_patch: { done: 'Edited article', running: 'Editing article' },
  replace_lines: { done: 'Edited article', running: 'Editing article' },
  generate_image_prompt: { done: 'Drafted an image prompt', running: 'Drafting an image prompt' },
  sandbox: { done: 'Ran command', running: 'Running command' },
  ask_question: { done: 'Answered', running: 'Answering' },
};

function toolActivity(call: ToolCallRecord): string {
  const labels = TOOL_ACTIVITY[call.name];
  const running = call.status === 'running' || call.status === 'pending';
  if (labels) return running ? labels.running : labels.done;
  return getToolDisplayName(call.name);
}

function insightFilters(input: Record<string, unknown>): string {
  const parts: string[] = [];
  const topic = stringField(input, 'topic');
  if (topic) parts.push(topic);
  if (input.unread === true) parts.push('unread');
  if (input.pinned === true) parts.push('pinned');
  if (input.unused === true) parts.push('unused');
  return parts.join(', ');
}

function toolDetail(call: ToolCallRecord): string {
  const input = call.input || {};
  const result = call.result || {};
  switch (call.name) {
    case 'list_insights':
    case 'search_insights': {
      const insights = Array.isArray(result.insights) ? result.insights : null;
      const count = insights == null
        ? ''
        : insights.length === 0
          ? 'none yet'
          : insights.length === 1
            ? '1 briefing'
            : `${insights.length} briefings`;
      const filters = insightFilters(input);
      const query = call.name === 'search_insights'
        ? (stringField(result, 'query') || stringField(input, 'query'))
        : '';
      return [query, filters, count].filter(Boolean).join(' · ');
    }
    case 'web_search':
    case 'search_web_sources':
    case 'deep_research':
    case 'get_relevant_sources':
      return stringField(result, 'query') || stringField(input, 'query');
    case 'set_title':
      return stringField(result, 'new_title') || stringField(input, 'title');
    case 'read_document': {
      const title = stringField(result, 'title');
      const lines = result.total_lines;
      if (title && typeof lines === 'number') return `${title} · ${lines} lines`;
      if (typeof lines === 'number') return `${lines} lines`;
      return title;
    }
    case 'update_sources': {
      const sources = Array.isArray(result.sources) ? result.sources.length : null;
      if (sources == null) return '';
      return sources === 1 ? '1 source' : `${sources} sources`;
    }
    case 'apply_patch':
    case 'replace_lines':
    case 'generate_image_prompt':
      return stringField(result, 'reason') || stringField(input, 'reason') || stringField(result, 'prompt');
    case 'sandbox':
      return stringField(input, 'command') || stringField(result, 'command');
    case 'ask_question':
      return stringField(input, 'question') || stringField(result, 'question');
    case 'select_sources_for_edit':
      return typeof result.selected_count === 'number' ? `${result.selected_count} selected` : '';
    default:
      return stringField(input, 'query') || stringField(input, 'title') || stringField(input, 'reason');
  }
}

/**
 * Compact tool row. The action and its subject stay on one line.
 * Open it to read the briefing, sources, or command output.
 */
function SubtleToolDisplay({ call }: { call: ToolCallRecord }) {
  const [isOpen, setIsOpen] = React.useState(false);
  const finished = call.status === 'completed' || call.status === 'error';
  const expandableContent = finished ? getExpandableContent(call) : null;
  const errorText = call.status === 'error' ? (call.error || 'Tool failed') : '';
  const hasContent = expandableContent !== null || Boolean(errorText);
  const detail = toolDetail(call);

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          if (hasContent) setIsOpen((prev) => !prev);
        }}
        disabled={!hasContent}
        className={cn(
          "flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-left text-[13px] text-muted-foreground",
          hasContent && "cursor-pointer hover:bg-muted/50 hover:text-foreground",
          !hasContent && "cursor-default"
        )}
      >
        <span className="flex size-4 shrink-0 items-center justify-center">
          {getToolIcon(call.name)}
        </span>
        <span className="min-w-0 flex-1 truncate">
          <span className="text-foreground/85">{toolActivity(call)}</span>
          {detail ? <span className="text-muted-foreground"> {detail}</span> : null}
        </span>
        <span className="flex shrink-0 items-center gap-1.5">
          {call.status === 'running' || call.status === 'pending' ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : call.status === 'error' ? (
            <X className="h-3.5 w-3.5 text-red-500" />
          ) : (
            <Check className="h-3.5 w-3.5" />
          )}
          {hasContent ? (
            <ChevronDown
              className={cn(
                "h-3.5 w-3.5 transition-transform duration-200",
                isOpen && "rotate-180"
              )}
            />
          ) : null}
        </span>
      </button>

      {isOpen && hasContent ? (
        <div className="scrollbar-subtle ml-6 max-h-64 overflow-y-auto border-l border-border/70 py-1 pl-2 text-xs text-muted-foreground">
          {errorText ? <p className="text-red-500">{errorText}</p> : expandableContent}
        </div>
      ) : null}
    </div>
  );
}

/**
 * Renders a group of tool calls using appropriate UI based on tool type
 */
export function ToolGroupDisplay({ group, onArtifactAction }: ToolGroupDisplayProps) {
  return (
    <div className="space-y-1">
      {group.calls.map((call) => {
        // Use DiffArtifact for edit tools (both completed and errored -- show whatever data is available)
        if (isArtifactTool(call.name) && (call.status === 'completed' || call.status === 'error') && (call.result || call.error)) {
          const result = call.result || {};
          // replace_lines uses old_str/new_str, rewrite_document uses original_content/new_content
          const oldText = (result.old_str || result.original_content || '') as string;
          const newText = (result.new_str || result.new_content || '') as string;
          const reason = (result.reason || '') as string;
          const errorMsg = call.error || (result.error as string) || (result.is_error ? 'Edit failed' : '');
          
          return (
            <DiffArtifact
              key={call.id}
              title={toolActivity(call)}
              description={reason}
              oldText={oldText}
              newText={newText}
              error={errorMsg || undefined}
              status={errorMsg ? 'error' : call.status === 'completed' ? 'completed' : 'error'}
              onApply={call.status === 'completed' && !errorMsg && onArtifactAction ? () => onArtifactAction(call.id, 'accept') : undefined}
            />
          );
        }
        
        return <SubtleToolDisplay key={call.id} call={call} />;
      })}
    </div>
  );
}

export default ToolGroupDisplay;
