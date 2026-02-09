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
 * Flat action buttons that appear above the chat input
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
      "flex items-center justify-end gap-1 px-3 py-1.5",
      className
    )}>
      <Button 
        variant="ghost"
        size="sm"
        onClick={onReject} 
        disabled={disabled}
        className="h-7 px-2.5 text-xs text-muted-foreground hover:text-foreground gap-1"
      >
        <X className="h-3.5 w-3.5" />
        Reject
      </Button>
      <Button 
        variant="ghost"
        size="sm"
        onClick={onKeepAll} 
        disabled={disabled}
        className="h-7 px-2.5 text-xs text-muted-foreground hover:text-foreground gap-1"
      >
        <Check className="h-3.5 w-3.5" />
        Keep all
      </Button>
    </div>
  )
}
