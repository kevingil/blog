import { useState } from "react"
import { ChevronDown, ChevronUp, FileEdit, FileDiff } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { ToolCallStatusItem } from "@/components/prompt-kit/tool-call"

interface DiffPreviewContentProps {
  toolName: 'edit_text' | 'rewrite_document'
  status: 'running' | 'completed' | 'error'
  reason?: string
  diffPreview?: {
    oldText: string
    newText: string
  }
  className?: string
}

const MAX_LINES = 5

export function DiffPreviewContent({
  toolName,
  status,
  reason,
  diffPreview,
  className
}: DiffPreviewContentProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  
  const isEditText = toolName === 'edit_text'

  // Check if content needs truncation
  const oldLines = diffPreview?.oldText.split('\n') || []
  const newLines = diffPreview?.newText.split('\n') || []
  const needsTruncation = oldLines.length > MAX_LINES || newLines.length > MAX_LINES
  
  // Get display text (truncated or full)
  const displayOldText = isExpanded ? diffPreview?.oldText : oldLines.slice(0, MAX_LINES).join('\n')
  const displayNewText = isExpanded ? diffPreview?.newText : newLines.slice(0, MAX_LINES).join('\n')
  const oldTruncated = !isExpanded && oldLines.length > MAX_LINES
  const newTruncated = !isExpanded && newLines.length > MAX_LINES

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
            <span className="text-red-600 dark:text-red-400">-{diffPreview.oldText.length}</span>
            <span>/</span>
            <span className="text-green-600 dark:text-green-400">+{diffPreview.newText.length}</span>
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

      {/* Simple old/new blocks */}
      {diffPreview && (
        <div className="space-y-2">
          {/* Old content - red */}
          {diffPreview.oldText && (
            <div>
              <div className="text-xs font-medium text-red-600 dark:text-red-400 mb-1">Removing:</div>
              <div className="bg-red-100 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded p-2 font-mono text-xs whitespace-pre-wrap text-red-800 dark:text-red-200 max-h-64 overflow-auto">
                {displayOldText}
                {oldTruncated && <span className="text-red-400 dark:text-red-500">...</span>}
              </div>
            </div>
          )}

          {/* New content - green */}
          {diffPreview.newText && (
            <div>
              <div className="text-xs font-medium text-green-600 dark:text-green-400 mb-1">Adding:</div>
              <div className="bg-green-100 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded p-2 font-mono text-xs whitespace-pre-wrap text-green-800 dark:text-green-200 max-h-64 overflow-auto">
                {displayNewText}
                {newTruncated && <span className="text-green-400 dark:text-green-500">...</span>}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Expand/collapse button */}
      {diffPreview && needsTruncation && (
        <div className="flex justify-center">
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
