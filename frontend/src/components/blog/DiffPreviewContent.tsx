import { useState } from "react"
import { ChevronDown, ChevronUp, FileEdit, FileDiff } from "lucide-react"
import { diffWords } from "diff"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { ToolCallStatusItem } from "@/components/prompt-kit/tool-call"

type DiffPart = { 
  added?: boolean
  removed?: boolean
  value: string
  truncated?: boolean 
}

interface DiffPreviewContentProps {
  toolName: 'edit_text' | 'rewrite_document'
  status: 'pending' | 'running' | 'completed' | 'error'
  reason?: string
  diffPreview?: {
    oldText: string
    newText: string
  }
  diffState?: 'accepted' | 'rejected'
  onAccept?: () => void
  onReject?: () => void
  className?: string
}

export function DiffPreviewContent({
  toolName,
  status,
  reason,
  diffPreview,
  diffState,
  onAccept,
  onReject,
  className
}: DiffPreviewContentProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  
  const isEditText = toolName === 'edit_text'
  const toolLabel = isEditText ? 'Text edit' : 'Document rewrite'

  // Compute diff parts if we have diff preview
  const parts = diffPreview ? diffWords(diffPreview.oldText, diffPreview.newText) : []
  const maxLines = 3

  // Truncate logic
  let currentLine = 0
  let truncatedParts: DiffPart[] = []
  let hasMoreContent = false

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]
    const lines = part.value.split('\n')

    if (currentLine + lines.length <= maxLines || isExpanded) {
      truncatedParts.push(part)
      currentLine += lines.length - 1
    } else {
      const remainingLines = maxLines - currentLine
      if (remainingLines > 0) {
        const truncatedValue = lines.slice(0, remainingLines).join('\n')
        truncatedParts.push({
          ...part,
          value: truncatedValue,
          truncated: true
        })
      }
      hasMoreContent = true
      break
    }
  }

  const showExpandButton = !isExpanded && (hasMoreContent || parts.some(p => p.value.split('\n').length > maxLines))

  // Calculate stats
  const addedCount = parts.filter(p => p.added).reduce((sum, p) => sum + p.value.length, 0)
  const removedCount = parts.filter(p => p.removed).reduce((sum, p) => sum + p.value.length, 0)

  return (
    <div className={cn("space-y-3", className)}>
      {/* Progress steps */}
      <div className="space-y-1">
        <ToolCallStatusItem status={status === 'running' ? 'running' : 'completed'}>
          {isEditText ? 'Analyzing text selection...' : 'Analyzing document...'}
        </ToolCallStatusItem>
        
        {status !== 'running' && (
          <ToolCallStatusItem status="completed">
            {isEditText ? 'Generated text edit' : 'Generated document rewrite'}
          </ToolCallStatusItem>
        )}
      </div>

      {/* Stats summary */}
      {diffPreview && (
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <span className="text-green-600 dark:text-green-400">+{addedCount}</span>
            <span>/</span>
            <span className="text-red-600 dark:text-red-400">-{removedCount}</span>
            <span>chars</span>
          </span>
          {reason && (
            <>
              <span>•</span>
              <span className="truncate">{reason}</span>
            </>
          )}
        </div>
      )}

      {/* Diff preview (when available) */}
      {diffPreview && diffState && (
        <div className="rounded-md border bg-muted/30 p-2">
          <div className={cn(
            "flex items-center gap-1.5 mb-2 text-xs font-medium",
            diffState === 'accepted' ? "text-green-700 dark:text-green-300" : "text-red-700 dark:text-red-300"
          )}>
            <span>{diffState === 'accepted' ? '✅' : '❌'}</span>
            <span>{diffState === 'accepted' ? 'Accepted' : 'Rejected'}</span>
          </div>
          
          <div className="font-mono text-xs whitespace-pre-wrap overflow-hidden">
            {(isExpanded ? parts : truncatedParts).map((part, index) => (
              <span
                key={index}
                className={cn(
                  part.added ? "bg-green-200 dark:bg-green-800 text-green-900 dark:text-green-100" :
                  part.removed ? "bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100 line-through" :
                  "text-muted-foreground"
                )}
              >
                {part.value}
                {'truncated' in part && part.truncated && <span className="text-muted-foreground/50">...</span>}
              </span>
            ))}
          </div>

          {(showExpandButton || isExpanded) && (
            <div className="flex justify-center mt-2">
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
        </div>
      )}

      {/* Action buttons (when pending) */}
      {status === 'pending' && onAccept && onReject && (
        <div className="flex gap-2">
          <Button size="sm" onClick={onAccept} className="flex-1">
            Accept
          </Button>
          <Button size="sm" variant="outline" onClick={onReject} className="flex-1">
            Discard
          </Button>
        </div>
      )}

      {/* Running state */}
      {status === 'running' && (
        <div className="text-xs text-muted-foreground animate-pulse">
          Processing changes...
        </div>
      )}

      {/* Error state */}
      {status === 'error' && (
        <div className="text-xs text-red-600 dark:text-red-400">
          Failed to apply changes
        </div>
      )}
    </div>
  )
}

// Helper to get the appropriate icon for diff tools
export function getDiffToolIcon(toolName: 'edit_text' | 'rewrite_document') {
  return toolName === 'edit_text' 
    ? <FileEdit className="h-4 w-4" />
    : <FileDiff className="h-4 w-4" />
}

// Helper to get display name for diff tools
export function getDiffToolDisplayName(toolName: 'edit_text' | 'rewrite_document', reason?: string) {
  const baseName = toolName === 'edit_text' ? 'Text edit' : 'Document rewrite'
  return reason ? `${baseName}: ${reason}` : baseName
}
