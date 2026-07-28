"use client"

import * as React from "react"
import { ChevronDown, Wrench, Loader2, CheckCircle2, XCircle, Clock } from "lucide-react"
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
      <div className={cn("w-full flex justify-start", className)}>
        <div className="max-w-lg w-full rounded-lg border bg-card text-card-foreground shadow-sm">
          {children}
        </div>
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

  const getStatusIcon = () => {
    switch (status) {
      case 'pending':
        return <Clock className="h-3.5 w-3.5 text-yellow-500" />
      case 'running':
        return <Loader2 className="h-3.5 w-3.5 text-blue-500 animate-spin" />
      case 'completed':
        return <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
      case 'error':
        return <XCircle className="h-3.5 w-3.5 text-red-500" />
      default:
        return null
    }
  }

  const getStatusBadge = () => {
    const baseClasses = "text-xs px-1.5 py-0.5 rounded-full font-medium"
    switch (status) {
      case 'pending':
        return (
          <span className={cn(baseClasses, "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400")}>
            Ready
          </span>
        )
      case 'running':
        return (
          <span className={cn(baseClasses, "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400")}>
            Processing
          </span>
        )
      case 'completed':
        return (
          <span className={cn(baseClasses, "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400")}>
            Completed
          </span>
        )
      case 'error':
        return (
          <span className={cn(baseClasses, "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400")}>
            Error
          </span>
        )
      default:
        return null
    }
  }

  return (
    <button
      onClick={toggleOpen}
      className={cn(
        "flex w-full items-center justify-between px-3 py-2.5 text-left text-sm",
        "hover:bg-muted/50 transition-colors",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        isOpen && "border-b",
        className
      )}
    >
      <span className="flex items-center gap-2 min-w-0 flex-1">
        <span className="flex-shrink-0 text-muted-foreground">
          {icon || <Wrench className="h-4 w-4" />}
        </span>
        <span className="truncate font-medium">{children}</span>
        {getStatusBadge()}
      </span>
      <span className="flex items-center gap-2 flex-shrink-0 ml-2">
        {getStatusIcon()}
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

interface ToolCallContentProps {
  children: React.ReactNode
  className?: string
}

export function ToolCallContent({ children, className }: ToolCallContentProps) {
  const { isOpen } = useToolCallContext()

  if (!isOpen) return null

  return (
    <div className={cn("px-3 py-3 space-y-2 text-sm", className)}>
      {children}
    </div>
  )
}

// Helper component for displaying tool input/output
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
      <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        {label}
      </div>
      <pre className="text-xs bg-muted/50 rounded-md p-2 overflow-x-auto whitespace-pre-wrap break-words">
        {displayData}
      </pre>
    </div>
  )
}

// Status item component for showing progress steps within tool content
interface ToolCallStatusItemProps {
  children: React.ReactNode
  status?: 'pending' | 'running' | 'completed' | 'error'
  className?: string
}

export function ToolCallStatusItem({ children, status = 'completed', className }: ToolCallStatusItemProps) {
  const getIcon = () => {
    switch (status) {
      case 'pending':
        return <div className="h-2 w-2 rounded-full bg-gray-400" />
      case 'running':
        return <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
      case 'completed':
        return <div className="h-2 w-2 rounded-full bg-green-500" />
      case 'error':
        return <div className="h-2 w-2 rounded-full bg-red-500" />
      default:
        return <div className="h-2 w-2 rounded-full bg-gray-400" />
    }
  }

  return (
    <div className={cn("flex items-center gap-2 text-muted-foreground", className)}>
      {getIcon()}
      <span>{children}</span>
    </div>
  )
}
