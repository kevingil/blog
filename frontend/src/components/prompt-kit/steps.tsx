"use client"

import * as React from "react"
import { ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"

interface StepsContextValue {
  isOpen: boolean
  toggleOpen: () => void
}

const StepsContext = React.createContext<StepsContextValue | undefined>(undefined)

function useStepsContext() {
  const context = React.useContext(StepsContext)
  if (!context) {
    throw new Error("Steps components must be used within Steps")
  }
  return context
}

interface StepsProps {
  children: React.ReactNode
  defaultOpen?: boolean
  className?: string
}

export function Steps({ children, defaultOpen = false, className }: StepsProps) {
  const [isOpen, setIsOpen] = React.useState(defaultOpen)

  const toggleOpen = React.useCallback(() => {
    setIsOpen((prev) => !prev)
  }, [])

  return (
    <StepsContext.Provider value={{ isOpen, toggleOpen }}>
      <div className={cn("w-full flex justify-start", className)}>
        <div className="max-w-lg w-full rounded-lg border bg-card text-card-foreground shadow-sm">
          {children}
        </div>
      </div>
    </StepsContext.Provider>
  )
}

interface StepsTriggerProps {
  children: React.ReactNode
  className?: string
}

export function StepsTrigger({ children, className }: StepsTriggerProps) {
  const { isOpen, toggleOpen } = useStepsContext()

  return (
    <button
      onClick={toggleOpen}
      className={cn(
        "flex w-full items-center justify-between px-4 py-3 text-left text-sm font-medium",
        "hover:bg-muted/50 transition-colors",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        isOpen && "border-b",
        className
      )}
    >
      <span className="flex items-center gap-2">
        <span className="text-primary">🔍</span>
        {children}
      </span>
      <ChevronDown
        className={cn(
          "h-4 w-4 transition-transform duration-200",
          isOpen && "rotate-180"
        )}
      />
    </button>
  )
}

interface StepsContentProps {
  children: React.ReactNode
  className?: string
}

export function StepsContent({ children, className }: StepsContentProps) {
  const { isOpen } = useStepsContext()

  if (!isOpen) return null

  return (
    <div className={cn("px-4 py-3 space-y-2 text-sm", className)}>
      {children}
    </div>
  )
}

interface StepsItemProps {
  children: React.ReactNode
  status?: "starting" | "running" | "completed" | "error"
  className?: string
}

export function StepsItem({ children, status = "completed", className }: StepsItemProps) {
  const getIcon = () => {
    switch (status) {
      case "starting":
      case "running":
        return (
          <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
        )
      case "completed":
        return (
          <div className="h-2 w-2 rounded-full bg-green-500" />
        )
      case "error":
        return (
          <div className="h-2 w-2 rounded-full bg-red-500" />
        )
      default:
        return (
          <div className="h-2 w-2 rounded-full bg-gray-400" />
        )
    }
  }

  return (
    <div className={cn("flex items-center gap-2 text-muted-foreground", className)}>
      {getIcon()}
      <span>{children}</span>
    </div>
  )
}

