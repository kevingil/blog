"use client"

import * as React from "react"
import { ChevronDown, Brain, Wrench, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"

// =============================================================================
// ChainOfThought - Container for a sequence of steps
// =============================================================================

interface ChainOfThoughtProps {
  children: React.ReactNode
  className?: string
}

export function ChainOfThought({ children, className }: ChainOfThoughtProps) {
  return (
    <div className={cn("space-y-0", className)}>
      {children}
    </div>
  )
}

// =============================================================================
// ChainOfThoughtStep - Individual step with optional vertical connector
// =============================================================================

type StepType = 'reasoning' | 'tool' | 'content'
type StepStatus = 'pending' | 'running' | 'completed'

interface ChainOfThoughtStepContextValue {
  isOpen: boolean
  toggleOpen: () => void
  type: StepType
  status: StepStatus
  isStreaming: boolean
}

const ChainOfThoughtStepContext = React.createContext<ChainOfThoughtStepContextValue | undefined>(undefined)

function useChainOfThoughtStepContext() {
  const context = React.useContext(ChainOfThoughtStepContext)
  if (!context) {
    throw new Error("ChainOfThoughtStep components must be used within ChainOfThoughtStep")
  }
  return context
}

interface ChainOfThoughtStepProps {
  children: React.ReactNode
  type?: StepType
  status?: StepStatus
  isLast?: boolean
  isStreaming?: boolean
  defaultOpen?: boolean
  className?: string
}

export function ChainOfThoughtStep({ 
  children, 
  type = 'content',
  status = 'completed',
  isLast = false, 
  isStreaming = false,
  defaultOpen,
  className 
}: ChainOfThoughtStepProps) {
  // Auto-open during streaming or when running, auto-close when done
  const [isOpen, setIsOpen] = React.useState(() => {
    if (defaultOpen !== undefined) return defaultOpen
    return isStreaming || status === 'running'
  })

  // Auto-collapse when streaming ends
  React.useEffect(() => {
    if (!isStreaming && status === 'completed' && type === 'reasoning') {
      // Give a small delay before collapsing
      const timer = setTimeout(() => {
        setIsOpen(false)
      }, 300)
      return () => clearTimeout(timer)
    }
  }, [isStreaming, status, type])

  const toggleOpen = React.useCallback(() => {
    setIsOpen((prev) => !prev)
  }, [])

  return (
    <ChainOfThoughtStepContext.Provider value={{ isOpen, toggleOpen, type, status, isStreaming }}>
      <div className={cn("relative", className)}>
        {/* Vertical connector line */}
        {!isLast && (
          <div className="absolute left-[15px] top-[32px] bottom-0 w-px bg-border" />
        )}
        <div className="relative">
          {children}
        </div>
      </div>
    </ChainOfThoughtStepContext.Provider>
  )
}

// =============================================================================
// ChainOfThoughtTrigger - Clickable header for collapsible steps
// =============================================================================

interface ChainOfThoughtTriggerProps {
  children: React.ReactNode
  icon?: React.ReactNode
  badge?: React.ReactNode
  className?: string
}

export function ChainOfThoughtTrigger({ 
  children, 
  icon,
  badge,
  className 
}: ChainOfThoughtTriggerProps) {
  const { isOpen, toggleOpen, type, status, isStreaming } = useChainOfThoughtStepContext()

  const getDefaultIcon = () => {
    switch (type) {
      case 'reasoning':
        return <Brain className="h-4 w-4" />
      case 'tool':
        return <Wrench className="h-4 w-4" />
      default:
        return null
    }
  }

  const getStatusIndicator = () => {
    if (isStreaming || status === 'running') {
      return <Loader2 className="h-3.5 w-3.5 text-purple-500 animate-spin" />
    }
    return null
  }

  const displayIcon = icon || getDefaultIcon()

  return (
    <button
      onClick={toggleOpen}
      className={cn(
        "flex w-full items-center gap-2 px-3 py-2 text-left text-sm",
        "hover:bg-muted/50 transition-colors rounded-lg",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        className
      )}
    >
      {displayIcon && (
        <span className="flex-shrink-0 text-muted-foreground">
          {displayIcon}
        </span>
      )}
      <span className="flex-1 min-w-0 flex items-center gap-2">
        <span className="truncate font-medium">{children}</span>
        {badge}
      </span>
      <span className="flex items-center gap-2 flex-shrink-0">
        {getStatusIndicator()}
        <ChevronDown
          className={cn(
            "h-4 w-4 text-muted-foreground transition-transform duration-200",
            isOpen && "rotate-180"
          )}
        />
      </span>
    </button>
  )
}

// =============================================================================
// ChainOfThoughtContent - Expandable content area
// =============================================================================

interface ChainOfThoughtContentProps {
  children: React.ReactNode
  className?: string
}

export function ChainOfThoughtContent({ children, className }: ChainOfThoughtContentProps) {
  const { isOpen } = useChainOfThoughtStepContext()

  if (!isOpen) return null

  return (
    <div className={cn(
      "ml-6 pl-4 border-l-2 border-muted text-sm text-muted-foreground",
      "animate-in fade-in-0 slide-in-from-top-1 duration-200",
      className
    )}>
      <div className="py-2 whitespace-pre-wrap">
        {children}
      </div>
    </div>
  )
}

// =============================================================================
// ChainOfThoughtItem - Non-collapsible inline item (for text content)
// =============================================================================

interface ChainOfThoughtItemProps {
  children: React.ReactNode
  className?: string
}

export function ChainOfThoughtItem({ children, className }: ChainOfThoughtItemProps) {
  return (
    <div className={cn("py-1", className)}>
      {children}
    </div>
  )
}

// =============================================================================
// ReasoningStep - Pre-built reasoning step component
// =============================================================================

interface ReasoningStepProps {
  content: string
  isStreaming?: boolean
  durationMs?: number
  isLast?: boolean
  className?: string
}

export function ReasoningStep({ 
  content, 
  isStreaming = false, 
  durationMs,
  isLast = false,
  className 
}: ReasoningStepProps) {
  const status = isStreaming ? 'running' : 'completed'

  return (
    <ChainOfThoughtStep 
      type="reasoning" 
      status={status}
      isStreaming={isStreaming}
      isLast={isLast}
      className={className}
    >
      <ChainOfThoughtTrigger
        badge={
          durationMs != null && !isStreaming ? (
            <span className="text-xs px-1.5 py-0.5 rounded-full font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
              {Math.round(durationMs)}ms
            </span>
          ) : undefined
        }
      >
        {isStreaming ? "Reasoning..." : "Reasoning"}
      </ChainOfThoughtTrigger>
      <ChainOfThoughtContent>
        {content}
      </ChainOfThoughtContent>
    </ChainOfThoughtStep>
  )
}
