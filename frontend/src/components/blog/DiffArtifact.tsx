import { useState, useMemo, memo } from "react"
import { ChevronDown, ChevronUp, FileDiff, Check } from "lucide-react"
import { diffWords } from "diff"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { 
  ToolCall, 
  ToolCallTrigger, 
  ToolCallContent 
} from "@/components/prompt-kit/tool-call"

type DiffPart = { 
  added?: boolean
  removed?: boolean
  value: string
  truncated?: boolean 
}

interface DiffArtifactProps {
  title: string
  description?: string
  oldText: string
  newText: string
  onApply?: () => void
  className?: string
}

const MAX_LINES = 3

/**
 * Unified diff artifact component for displaying diff previews in chat.
 * Memoized to prevent expensive diffWords() recomputation on parent re-renders.
 * Works for any tool that creates diffs (edit_text, rewrite_document, etc.)
 */
export const DiffArtifact = memo(function DiffArtifact({
  title,
  description,
  oldText,
  newText,
  onApply,
  className
}: DiffArtifactProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  // Memoize the expensive diff computation - only recompute when text changes
  const parts = useMemo(
    () => diffWords(oldText, newText),
    [oldText, newText]
  )

  // Memoize derived values that depend on parts
  const { truncatedParts, hasMoreContent, addedCount, removedCount } = useMemo(() => {
    let currentLine = 0
    const truncated: DiffPart[] = []
    let hasMore = false

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      const lines = part.value.split('\n')

      if (currentLine + lines.length <= MAX_LINES) {
        truncated.push(part)
        currentLine += lines.length - 1
      } else {
        const remainingLines = MAX_LINES - currentLine
        if (remainingLines > 0) {
          const truncatedValue = lines.slice(0, remainingLines).join('\n')
          truncated.push({
            ...part,
            value: truncatedValue,
            truncated: true
          })
        }
        hasMore = true
        break
      }
    }

    const added = parts.filter(p => p.added).reduce((sum, p) => sum + p.value.length, 0)
    const removed = parts.filter(p => p.removed).reduce((sum, p) => sum + p.value.length, 0)

    return {
      truncatedParts: truncated,
      hasMoreContent: hasMore,
      addedCount: added,
      removedCount: removed
    }
  }, [parts])

  const showExpandButton = !isExpanded && (hasMoreContent || parts.some(p => p.value.split('\n').length > MAX_LINES))
  const hasDiffData = oldText.length > 0 || newText.length > 0

  return (
    <ToolCall defaultOpen={false} className={className}>
      <ToolCallTrigger icon={<FileDiff className="h-4 w-4" />}>
        {title}
      </ToolCallTrigger>
      <ToolCallContent>
        {/* Description */}
        {description && (
          <p className="text-xs text-muted-foreground mb-2">{description}</p>
        )}

        {/* Stats summary - only show when we have diff data */}
        {hasDiffData && (
          <div className="flex items-center gap-3 text-xs text-muted-foreground mb-2">
            <span className="flex items-center gap-1">
              <span className="text-green-600 dark:text-green-400">+{addedCount}</span>
              <span>/</span>
              <span className="text-red-600 dark:text-red-400">-{removedCount}</span>
              <span>chars</span>
            </span>
          </div>
        )}

        {/* Expandable diff preview - only show when we have diff data */}
        {hasDiffData && (
          <div className="rounded-md border bg-muted/30 p-2 mb-3">
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

        {/* Apply button - always visible when onApply is provided */}
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
  // Custom comparison: only re-render if data props change, not callbacks
  // This prevents re-renders when parent recreates onApply inline functions
  return (
    prevProps.oldText === nextProps.oldText &&
    prevProps.newText === nextProps.newText &&
    prevProps.title === nextProps.title &&
    prevProps.description === nextProps.description &&
    prevProps.className === nextProps.className
  )
})
