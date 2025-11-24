import { cn } from "@/lib/utils";
import React from "react";

interface ThinkShimmerProps {
  message?: string;
  className?: string;
}

/**
 * ThinkShimmer component - Animated shimmer effect for thinking/loading states
 * Inspired by https://www.prompt-kit.com/docs/loader
 */
export function ThinkShimmer({ message, className }: ThinkShimmerProps) {
  return (
    <div className={cn("flex items-center gap-3 py-2", className)}>
      {/* Animated shimmer dots */}
      <div className="flex items-center gap-1.5">
        <div
          className="h-2 w-2 rounded-full bg-primary/60 animate-pulse"
          style={{ animationDelay: "0ms", animationDuration: "1.4s" }}
        />
        <div
          className="h-2 w-2 rounded-full bg-primary/60 animate-pulse"
          style={{ animationDelay: "200ms", animationDuration: "1.4s" }}
        />
        <div
          className="h-2 w-2 rounded-full bg-primary/60 animate-pulse"
          style={{ animationDelay: "400ms", animationDuration: "1.4s" }}
        />
      </div>

      {/* Optional message */}
      {message && (
        <span className="text-sm text-muted-foreground animate-pulse">
          {message}
        </span>
      )}
    </div>
  );
}

interface ThinkShimmerBlockProps {
  message?: string;
  className?: string;
}

/**
 * ThinkShimmerBlock - Full shimmer block for chat messages
 */
export function ThinkShimmerBlock({ message = "Thinking...", className }: ThinkShimmerBlockProps) {
  return (
    <div className={cn("w-full flex justify-start", className)}>
      <div className="max-w-xs rounded-lg px-3 py-2 bg-muted/50 border border-muted">
        <ThinkShimmer message={message} />
      </div>
    </div>
  );
}

/**
 * ShimmerLine - Single line shimmer effect for content loading
 */
export function ShimmerLine({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "h-4 bg-gradient-to-r from-muted via-muted-foreground/20 to-muted rounded animate-shimmer",
        className
      )}
      style={{
        backgroundSize: "200% 100%",
        animation: "shimmer 2s ease-in-out infinite",
      }}
    />
  );
}

// Add shimmer animation to global CSS if not already present
// @keyframes shimmer {
//   0% { background-position: -200% 0; }
//   100% { background-position: 200% 0; }
// }

