"use client"

import * as React from "react"
import { ChevronUp, Loader2 } from "lucide-react"
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
          <div className="absolute left-[5px] top-[22px] bottom-0 w-px bg-border/50" />
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

  const getStatusIndicator = () => {
    if (isStreaming || status === 'running') {
      return <Loader2 className="h-3 w-3 text-muted-foreground animate-spin" />
    }
    return null
  }

  // Simple bullet point icon
  const bulletIcon = icon || (
    <span className="text-muted-foreground/70 text-xs">•</span>
  )

  return (
    <button
      onClick={toggleOpen}
      className={cn(
        "flex w-full items-center gap-2 py-1 text-left text-sm",
        "hover:text-foreground transition-colors",
        "focus:outline-none",
        "text-muted-foreground",
        className
      )}
    >
      <span className="flex-shrink-0 w-3 flex justify-center">
        {bulletIcon}
      </span>
      <span className="flex-1 min-w-0 flex items-center gap-2">
        <span className="truncate">{children}</span>
        {badge}
        {getStatusIndicator()}
      </span>
      <ChevronUp
        className={cn(
          "h-4 w-4 text-muted-foreground/50 transition-transform duration-200 flex-shrink-0",
          !isOpen && "rotate-180"
        )}
      />
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
      "ml-5 pl-3 border-l border-border/50 text-sm text-muted-foreground",
      "animate-in fade-in-0 slide-in-from-top-1 duration-200",
      className
    )}>
      <div className="py-1.5 whitespace-pre-wrap leading-relaxed">
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
    <div className={cn("py-1 ml-5", className)}>
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
      <ChainOfThoughtTrigger>
        {isStreaming ? "Reasoning..." : "Reasoning"}
      </ChainOfThoughtTrigger>
      <ChainOfThoughtContent>
        {content}
      </ChainOfThoughtContent>
    </ChainOfThoughtStep>
  )
}
