import { ToolGroupDisplay } from "./ToolGroupDisplay";
import { DiffArtifact } from "./DiffArtifact";
import { Markdown } from "@/components/ui/markdown";
import {
  ChainOfThought,
  ChainOfThoughtStep,
  ChainOfThoughtTrigger,
  ChainOfThoughtContent,
  ChainOfThoughtItem,
} from "@/components/prompt-kit/chain-of-thought";
import type { 
  AgentTurn, 
  Artifact, 
  MessageMetaData,
  TurnStep,
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
 * Renders a single step in the chain of thought
 */
function StepRenderer({ 
  step, 
  isLast,
  onArtifactAction,
  onApplyDiff,
}: { 
  step: TurnStep; 
  isLast: boolean;
  onArtifactAction?: (artifactId: string, action: 'accept' | 'reject') => void;
  onApplyDiff?: (oldText: string, newText: string, reason?: string) => void;
}) {
  switch (step.type) {
    case 'reasoning':
      if (!step.thinking?.content) return null;
      return (
        <ChainOfThoughtStep 
          type="reasoning" 
          status={step.isStreaming ? 'running' : 'completed'}
          isStreaming={step.isStreaming}
          isLast={isLast}
        >
          <ChainOfThoughtTrigger
            badge={
              step.thinking.duration_ms != null && !step.isStreaming ? (
                <span className="text-xs px-1.5 py-0.5 rounded-full font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                  {Math.round(step.thinking.duration_ms)}ms
                </span>
              ) : undefined
            }
          >
            {step.isStreaming ? "Reasoning..." : "Reasoning"}
          </ChainOfThoughtTrigger>
          <ChainOfThoughtContent>
            {step.thinking.content}
          </ChainOfThoughtContent>
        </ChainOfThoughtStep>
      );
    
    case 'tool_group':
      if (!step.toolGroup) return null;
      return (
        <ChainOfThoughtStep type="tool" status="completed" isLast={isLast}>
          <ToolGroupDisplay group={step.toolGroup} />
        </ChainOfThoughtStep>
      );
    
    case 'content':
      if (!step.content) return null;
      return (
        <ChainOfThoughtItem>
          <div className="prose prose-sm max-w-none dark:prose-invert">
            <Markdown>{step.content}</Markdown>
          </div>
        </ChainOfThoughtItem>
      );
    
    default:
      return null;
  }
}

/**
 * Unified component that renders an agent turn properly
 */
export function AgentMessage({ turn, onArtifactAction, onApplyDiff }: AgentMessageProps) {
  // If turn has steps array, use the new chain-of-thought rendering
  if (turn.steps && turn.steps.length > 0) {
    return (
      <div className="space-y-3">
        <ChainOfThought>
          {turn.steps.map((step, idx) => (
            <StepRenderer 
              key={idx} 
              step={step}
              isLast={idx === turn.steps!.length - 1}
              onArtifactAction={onArtifactAction}
              onApplyDiff={onApplyDiff}
            />
          ))}
        </ChainOfThought>
        
        {/* Artifacts (rendered outside chain) */}
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
  
  // Legacy rendering - build steps from individual fields
  const legacySteps: TurnStep[] = [];
  
  // Add reasoning step if present
  if (turn.thinking?.content) {
    legacySteps.push({
      type: 'reasoning',
      thinking: turn.thinking,
      isStreaming: turn.isReasoningStreaming,
    });
  }
  
  // Add tool group step if present
  if (turn.toolGroup) {
    legacySteps.push({
      type: 'tool_group',
      toolGroup: turn.toolGroup,
    });
  }
  
  // Add content step if present
  if (turn.content) {
    legacySteps.push({
      type: 'content',
      content: turn.content,
    });
  }
  
  // If we have steps, render with chain-of-thought
  if (legacySteps.length > 0) {
    return (
      <div className="space-y-3">
        <ChainOfThought>
          {legacySteps.map((step, idx) => (
            <StepRenderer 
              key={idx} 
              step={step}
              isLast={idx === legacySteps.length - 1}
              onArtifactAction={onArtifactAction}
              onApplyDiff={onApplyDiff}
            />
          ))}
        </ChainOfThought>
        
        {/* Artifacts (rendered outside chain) */}
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
  
  // Fallback: just render artifacts if nothing else
  if (turn.artifacts && turn.artifacts.length > 0) {
    return (
      <div className="space-y-3">
        {turn.artifacts.map((artifact) => (
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
  
  return null;
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
