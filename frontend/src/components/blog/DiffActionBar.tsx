import { Button } from "@/components/ui/button"
import { Check, X } from "lucide-react"
import { cn } from "@/lib/utils"

interface DiffActionBarProps {
  onKeepAll: () => void
  onReject: () => void
  disabled?: boolean
  className?: string
}

/**
 * Sticky action bar that appears at the bottom of the chat
 * when there are pending diff changes to accept or reject.
 */
export function DiffActionBar({
  onKeepAll,
  onReject,
  disabled = false,
  className
}: DiffActionBarProps) {
  return (
    <div className={cn(
      "sticky bottom-0 left-0 right-0 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-t p-3",
      className
    )}>
      <div className="flex items-center justify-between gap-3">
        <div className="text-sm text-muted-foreground">
          Pending changes ready for review
        </div>
        <div className="flex gap-2">
          <Button 
            size="sm" 
            onClick={onKeepAll} 
            disabled={disabled}
            className="gap-1.5"
          >
            <Check className="h-4 w-4" />
            Keep All
          </Button>
          <Button 
            size="sm" 
            variant="outline" 
            onClick={onReject} 
            disabled={disabled}
            className="gap-1.5"
          >
            <X className="h-4 w-4" />
            Reject
          </Button>
        </div>
      </div>
    </div>
  )
}
