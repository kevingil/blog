"use client"

import * as React from "react"
import * as PopoverPrimitive from "@radix-ui/react-popover"
import { ExternalLink } from "lucide-react"
import { cn } from "@/lib/utils"

interface SourceContextValue {
  href: string
}

const SourceContext = React.createContext<SourceContextValue | undefined>(undefined)

function useSourceContext() {
  const context = React.useContext(SourceContext)
  if (!context) {
    throw new Error("Source components must be used within Source")
  }
  return context
}

interface SourceProps {
  children: React.ReactNode
  href: string
}

export function Source({ children, href }: SourceProps) {
  return (
    <SourceContext.Provider value={{ href }}>
      <PopoverPrimitive.Root>
        {children}
      </PopoverPrimitive.Root>
    </SourceContext.Provider>
  )
}

interface SourceTriggerProps {
  label: string
  showFavicon?: boolean
  className?: string
}

export function SourceTrigger({ label, showFavicon = false, className }: SourceTriggerProps) {
  const { href } = useSourceContext()
  const [faviconError, setFaviconError] = React.useState(false)

  // Extract domain for favicon
  const getFaviconUrl = (url: string) => {
    try {
      const domain = new URL(url).origin
      return `${domain}/favicon.ico`
    } catch {
      return null
    }
  }

  const faviconUrl = showFavicon ? getFaviconUrl(href) : null

  return (
    <PopoverPrimitive.Trigger asChild>
      <button
        className={cn(
          "inline-flex items-center gap-1.5 px-2 py-1 rounded-md",
          "text-xs font-medium",
          "bg-blue-50 dark:bg-blue-950/30",
          "border border-blue-200 dark:border-blue-800",
          "text-blue-700 dark:text-blue-300",
          "hover:bg-blue-100 dark:hover:bg-blue-950/50",
          "transition-colors",
          "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
          className
        )}
      >
        {showFavicon && faviconUrl && !faviconError && (
          <img
            src={faviconUrl}
            alt=""
            className="h-3 w-3"
            onError={() => setFaviconError(true)}
          />
        )}
        <span className="max-w-[200px] truncate">{label}</span>
        <ExternalLink className="h-3 w-3 opacity-50" />
      </button>
    </PopoverPrimitive.Trigger>
  )
}

interface SourceContentProps {
  title: string
  description?: string
  metadata?: {
    author?: string
    published?: string
    [key: string]: any
  }
  className?: string
}

export function SourceContent({ title, description, metadata, className }: SourceContentProps) {
  const { href } = useSourceContext()

  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        align="start"
        sideOffset={4}
        className={cn(
          "z-50 w-80 rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none",
          "data-[state=open]:animate-in data-[state=closed]:animate-out",
          "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
          "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
          "data-[side=bottom]:slide-in-from-top-2",
          "data-[side=left]:slide-in-from-right-2",
          "data-[side=right]:slide-in-from-left-2",
          "data-[side=top]:slide-in-from-bottom-2",
          className
        )}
      >
        <div className="space-y-2">
          <div>
            <h4 className="font-medium text-sm leading-tight">{title}</h4>
            {description && (
              <p className="text-xs text-muted-foreground mt-1 line-clamp-3">
                {description}
              </p>
            )}
          </div>

          {metadata && (
            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
              {metadata.author && (
                <span className="flex items-center gap-1">
                  <span className="font-medium">By:</span>
                  {metadata.author}
                </span>
              )}
              {metadata.published && (
                <span className="flex items-center gap-1">
                  <span className="font-medium">Published:</span>
                  {new Date(metadata.published).toLocaleDateString()}
                </span>
              )}
            </div>
          )}

          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline mt-2"
          >
            Visit source
            <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      </PopoverPrimitive.Content>
    </PopoverPrimitive.Portal>
  )
}

