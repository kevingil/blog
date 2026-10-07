"use client"

import * as React from "react"
import { ChevronDown, Wrench, Loader2, Check, X } from "lucide-react"
import { cn } from "@/lib/utils"

type ToolStatus = 'pending' | 'running' | 'completed' | 'error'

interface ToolCallContextValue {
  isOpen: boolean
  toggleOpen: () => void
  status: ToolStatus
}

const ToolCallContext = React.createContext<ToolCallContextValue | undefined>(undefined)

function useToolCallContext() {
  const context = React.useContext(ToolCallContext)
  if (!context) {
    throw new Error("ToolCall components must be used within ToolCall")
  }
  return context
}

interface ToolCallProps {
  children: React.ReactNode
  status?: ToolStatus
  defaultOpen?: boolean
  className?: string
}

export function ToolCall({
  children,
  status = 'completed',
  defaultOpen = false,
  className
}: ToolCallProps) {
  const [isOpen, setIsOpen] = React.useState(defaultOpen)

  const toggleOpen = React.useCallback(() => {
    setIsOpen((prev) => !prev)
  }, [])

  return (
    <ToolCallContext.Provider value={{ isOpen, toggleOpen, status }}>
      <div className={cn("w-full text-[13px]", className)}>
        {children}
      </div>
    </ToolCallContext.Provider>
  )
}

interface ToolCallTriggerProps {
  children: React.ReactNode
  icon?: React.ReactNode
  className?: string
}

export function ToolCallTrigger({ children, icon, className }: ToolCallTriggerProps) {
  const { isOpen, toggleOpen, status } = useToolCallContext()

  const statusIcon = () => {
    switch (status) {
      case 'pending':
      case 'running':
        return <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
      case 'error':
        return <X className="h-3.5 w-3.5 text-red-500" />
      default:
        return <Check className="h-3.5 w-3.5 text-muted-foreground" />
    }
  }

  return (
    <button
      type="button"
      onClick={toggleOpen}
      className={cn(
        "flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-left text-muted-foreground",
        "hover:bg-muted/50 hover:text-foreground",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        className
      )}
    >
      <span className="flex size-4 shrink-0 items-center justify-center text-muted-foreground">
        {icon || <Wrench className="h-3.5 w-3.5" />}
      </span>
      <span className="min-w-0 flex-1 truncate text-foreground/85">{children}</span>
      <span className="flex shrink-0 items-center gap-1.5">
        {statusIcon()}
        <ChevronDown
          className={cn(
            "h-3.5 w-3.5 text-muted-foreground transition-transform duration-200",
            isOpen && "rotate-180"
          )}
        />
      </span>
    </button>
  )
}

interface ToolCallContentProps {
  children: React.ReactNode
  className?: string
}

export function ToolCallContent({ children, className }: ToolCallContentProps) {
  const { isOpen } = useToolCallContext()

  if (!isOpen) return null

  return (
    <div className={cn("scrollbar-subtle ml-6 max-h-64 overflow-auto border-l border-border/70 py-1 pl-2 text-xs text-muted-foreground", className)}>
      {children}
    </div>
  )
}

interface ToolCallDataProps {
  label: string
  data: Record<string, unknown> | string | null | undefined
  className?: string
}

export function ToolCallData({ label, data, className }: ToolCallDataProps) {
  if (!data) return null

  const displayData = typeof data === 'string' ? data : JSON.stringify(data, null, 2)

  return (
    <div className={cn("space-y-1", className)}>
      <div className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </div>
      <pre className="scrollbar-subtle overflow-x-auto whitespace-pre-wrap break-words rounded-md bg-muted/40 p-2 font-mono text-xs">
        {displayData}
      </pre>
    </div>
  )
}

interface ToolCallStatusItemProps {
  children: React.ReactNode
  status?: 'pending' | 'running' | 'completed' | 'error'
  className?: string
}

export function ToolCallStatusItem({ children, status = 'completed', className }: ToolCallStatusItemProps) {
  return (
    <div className={cn("text-muted-foreground", status === 'error' && "text-red-500", className)}>
      {children}
    </div>
  )
}
