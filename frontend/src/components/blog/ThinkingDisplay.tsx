import { Brain, ChevronDown } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import type { ThinkingBlock } from "./types";

interface ThinkingDisplayProps {
  thinking: ThinkingBlock;
  defaultOpen?: boolean;
}

/**
 * Collapsible chain-of-thought reasoning display
 */
export function ThinkingDisplay({ thinking, defaultOpen = false }: ThinkingDisplayProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  if (!thinking.content) {
    return null;
  }

  return (
    <div className="w-full flex justify-start">
      <div className="max-w-lg w-full rounded-lg border bg-card text-card-foreground shadow-sm">
        <button
          onClick={() => setIsOpen(!isOpen)}
          className={cn(
            "flex w-full items-center justify-between px-3 py-2.5 text-left text-sm",
            "hover:bg-muted/50 transition-colors",
            "focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
            isOpen && "border-b"
          )}
        >
          <span className="flex items-center gap-2 min-w-0 flex-1">
            <span className="flex-shrink-0 text-muted-foreground">
              <Brain className="h-4 w-4" />
            </span>
            <span className="truncate font-medium">Reasoning</span>
            {thinking.duration_ms && (
              <span className="text-xs px-1.5 py-0.5 rounded-full font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                {Math.round(thinking.duration_ms)}ms
              </span>
            )}
          </span>
          <ChevronDown
            className={cn(
              "h-4 w-4 text-muted-foreground transition-transform duration-200",
              isOpen && "rotate-180"
            )}
          />
        </button>
        
        {isOpen && (
          <div className="px-3 py-3 text-sm">
            <div className="text-muted-foreground whitespace-pre-wrap">
              {thinking.content}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ThinkingDisplay;
