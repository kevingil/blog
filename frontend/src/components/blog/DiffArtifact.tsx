import { useState, memo } from "react"
import { ChevronDown, ChevronUp, FileDiff, Check, AlertCircle } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { 
  ToolCall, 
  ToolCallTrigger, 
  ToolCallContent 
} from "@/components/prompt-kit/tool-call"

interface DiffArtifactProps {
  title: string
  description?: string
  oldText: string
  newText: string
  error?: string
  status?: 'pending' | 'running' | 'completed' | 'error'
  onApply?: () => void
  className?: string
}

const MAX_LINES = 5

/**
 * Simple diff artifact component showing old (red) and new (green) content blocks.
 * Also renders an error banner when the edit failed, while still showing
 * whatever diff data is available.
 */
export const DiffArtifact = memo(function DiffArtifact({
  title,
  description,
  oldText,
  newText,
  error,
  status,
  onApply,
  className
}: DiffArtifactProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  const hasDiffData = oldText.length > 0 || newText.length > 0
  
  // Check if content needs truncation
  const oldLines = oldText.split('\n')
  const newLines = newText.split('\n')
  const needsTruncation = oldLines.length > MAX_LINES || newLines.length > MAX_LINES
  
  // Get display text (truncated or full)
  const displayOldText = isExpanded ? oldText : oldLines.slice(0, MAX_LINES).join('\n')
  const displayNewText = isExpanded ? newText : newLines.slice(0, MAX_LINES).join('\n')
  const oldTruncated = !isExpanded && oldLines.length > MAX_LINES
  const newTruncated = !isExpanded && newLines.length > MAX_LINES

  return (
    <ToolCall defaultOpen={false} status={error ? 'error' : status} className={className}>
      <ToolCallTrigger icon={<FileDiff className="h-4 w-4" />}>
        {title}
      </ToolCallTrigger>
      <ToolCallContent>
        {/* Error banner */}
        {error && (
          <div className="flex items-start gap-2 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-2 mb-2">
            <AlertCircle className="h-3.5 w-3.5 text-red-500 mt-0.5 flex-shrink-0" />
            <p className="text-xs text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}

        {/* Description */}
        {description && (
          <p className="text-xs text-muted-foreground mb-2">{description}</p>
        )}

        {/* Stats summary */}
        {hasDiffData && (
          <div className="mb-2 font-mono text-[11px]">
            {oldText ? <span className="text-red-600 dark:text-red-400">-{oldLines.length}</span> : null}
            {oldText && newText ? <span className="mx-1 text-muted-foreground">/</span> : null}
            {newText ? <span className="text-green-600 dark:text-green-400">+{newLines.length}</span> : null}
          </div>
        )}

        {hasDiffData && (
          <pre className="scrollbar-subtle mb-2 max-h-64 overflow-auto rounded-md bg-muted/40 p-2 font-mono text-xs leading-5">
            {displayOldText.split('\n').filter(() => displayOldText.length > 0).map((line, index) => (
              <div key={`old-${index}`} className="whitespace-pre-wrap text-red-700 dark:text-red-300">
                - {line}
                {oldTruncated && index === displayOldText.split('\n').length - 1 ? '…' : ''}
              </div>
            ))}
            {displayNewText.split('\n').filter(() => displayNewText.length > 0).map((line, index) => (
              <div key={`new-${index}`} className="whitespace-pre-wrap text-green-700 dark:text-green-300">
                + {line}
                {newTruncated && index === displayNewText.split('\n').length - 1 ? '…' : ''}
              </div>
            ))}
          </pre>
        )}

        {/* Expand/collapse button */}
        {needsTruncation && (
          <div className="flex justify-center mb-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsExpanded(!isExpanded)}
              className="h-6 px-2 text-xs"
            >
              {isExpanded ? (
                <>
                  <ChevronUp className="h-3 w-3 mr-1" />
                  Show less
                </>
              ) : (
                <>
                  <ChevronDown className="h-3 w-3 mr-1" />
                  Show more
                </>
              )}
            </Button>
          </div>
        )}

        {/* Apply button -- only shown for successful edits */}
        {onApply && (
          <Button size="sm" onClick={onApply} className="w-full gap-1.5">
            <Check className="h-4 w-4" />
            Apply
          </Button>
        )}
      </ToolCallContent>
    </ToolCall>
  )
}, (prevProps, nextProps) => {
  return (
    prevProps.oldText === nextProps.oldText &&
    prevProps.newText === nextProps.newText &&
    prevProps.title === nextProps.title &&
    prevProps.description === nextProps.description &&
    prevProps.error === nextProps.error &&
    prevProps.status === nextProps.status &&
    prevProps.className === nextProps.className
  )
})
