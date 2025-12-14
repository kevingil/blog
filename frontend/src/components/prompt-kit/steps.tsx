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
  icon?: React.ReactNode
  className?: string
}

export function StepsTrigger({ children, icon, className }: StepsTriggerProps) {
  const { isOpen, toggleOpen } = useStepsContext()

  // Default icon is a wrench for tools
  const defaultIcon = (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-4 w-4"
    >
      <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
    </svg>
  )

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
        <span className="text-muted-foreground flex-shrink-0">
          {icon || defaultIcon}
        </span>
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

