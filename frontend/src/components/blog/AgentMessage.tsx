import { ThinkingDisplay } from "./ThinkingDisplay";
import { ToolGroupDisplay } from "./ToolGroupDisplay";
import { DiffArtifact } from "./DiffArtifact";
import { Markdown } from "@/components/ui/markdown";
import type { 
  AgentTurn, 
  Artifact, 
  MessageMetaData,
} from "./types";

interface AgentMessageProps {
  turn: AgentTurn;
  onArtifactAction?: (artifactId: string, action: 'accept' | 'reject') => void;
  onApplyDiff?: (oldText: string, newText: string, reason?: string) => void;
}

/**
 * Renders an artifact based on its type
 */
function ArtifactDisplay({ 
  artifact, 
  onAction,
  onApplyDiff,
}: { 
  artifact: Artifact; 
  onAction?: (action: 'accept' | 'reject') => void;
  onApplyDiff?: (oldText: string, newText: string, reason?: string) => void;
}) {
  switch (artifact.type) {
    case 'diff':
      const diffData = artifact.data as { original?: string; proposed?: string; reason?: string };
      return (
        <DiffArtifact
          title="Suggested changes"
          description={diffData.reason}
          oldText={diffData.original || ''}
          newText={diffData.proposed || ''}
          onApply={() => {
            onAction?.('accept');
            onApplyDiff?.(diffData.original || '', diffData.proposed || '', diffData.reason);
          }}
        />
      );
    
    case 'sources':
      // Sources are rendered inline in ToolGroupDisplay
      return null;
    
    case 'answer':
      // Answers are rendered inline in ToolGroupDisplay
      return null;
    
    case 'content_generation':
      const contentData = artifact.data as { generated_content?: string };
      return (
        <div className="bg-muted/50 rounded-lg p-4 border">
          <h4 className="text-sm font-medium mb-2">Generated Content</h4>
          <div className="text-sm text-muted-foreground">
            <Markdown>{contentData.generated_content || ''}</Markdown>
          </div>
        </div>
      );
    
    case 'image_prompt':
      const promptData = artifact.data as { prompt?: string };
      return (
        <div className="bg-muted/50 rounded-lg p-4 border">
          <h4 className="text-sm font-medium mb-2">Image Prompt</h4>
          <div className="text-sm text-muted-foreground font-mono">
            {promptData.prompt}
          </div>
        </div>
      );
    
    default:
      return null;
  }
}

/**
 * Unified component that renders an agent turn properly
 */
export function AgentMessage({ turn, onArtifactAction, onApplyDiff }: AgentMessageProps) {
  return (
    <div className="space-y-3">
      {/* Chain of thought / thinking */}
      {turn.thinking && (
        <ThinkingDisplay thinking={turn.thinking} />
      )}
      
      {/* Tool execution group */}
      {turn.toolGroup && (
        <ToolGroupDisplay group={turn.toolGroup} />
      )}
      
      {/* Text content */}
      {turn.content && (
        <div className="prose prose-sm max-w-none dark:prose-invert">
          <Markdown>{turn.content}</Markdown>
        </div>
      )}
      
      {/* Artifacts */}
      {turn.artifacts?.map((artifact) => (
        <ArtifactDisplay
          key={artifact.id}
          artifact={artifact}
          onAction={(action) => onArtifactAction?.(artifact.id, action)}
          onApplyDiff={onApplyDiff}
        />
      ))}
    </div>
  );
}

/**
 * Helper function to convert legacy message metadata to AgentTurn
 */
export function convertMetaDataToTurn(
  messageId: string,
  content: string,
  metaData?: MessageMetaData,
  createdAt?: string
): AgentTurn {
  const turn: AgentTurn = {
    id: messageId,
    turn_sequence: metaData?.turn_sequence || 0,
    content: content,
    created_at: createdAt || new Date().toISOString(),
  };

  // Convert thinking
  if (metaData?.thinking) {
    turn.thinking = metaData.thinking;
  }

  // Convert tool group
  if (metaData?.tool_group) {
    turn.toolGroup = metaData.tool_group;
  }

  // Convert artifacts
  if (metaData?.artifacts && metaData.artifacts.length > 0) {
    turn.artifacts = metaData.artifacts;
  }

  // Handle legacy tool_execution format
  if (metaData?.tool_execution && !metaData.tool_group) {
    const toolExec = metaData.tool_execution;
    turn.toolGroup = {
      group_id: toolExec.tool_id,
      status: toolExec.success ? 'completed' : 'error',
      calls: [{
        id: toolExec.tool_id,
        name: toolExec.tool_name,
        input: typeof toolExec.input === 'object' ? toolExec.input as Record<string, unknown> : {},
        status: toolExec.success ? 'completed' : 'error',
        result: typeof toolExec.output === 'object' ? toolExec.output as Record<string, unknown> : undefined,
        error: toolExec.error,
        started_at: toolExec.executed_at,
        duration_ms: toolExec.duration_ms,
      }],
    };
  }

  // Handle legacy artifact format
  if (metaData?.artifact && !metaData.artifacts) {
    const legacyArtifact = metaData.artifact;
    turn.artifacts = [{
      id: legacyArtifact.id,
      type: legacyArtifact.type as 'diff' | 'sources' | 'answer' | 'content_generation' | 'image_prompt',
      status: legacyArtifact.status as 'pending' | 'accepted' | 'rejected',
      data: {
        content: legacyArtifact.content,
        diff_preview: legacyArtifact.diff_preview,
        title: legacyArtifact.title,
        description: legacyArtifact.description,
      },
    }];
  }

  return turn;
}

export default AgentMessage;
