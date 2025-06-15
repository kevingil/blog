import * as React from "react"
import { X } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface ChipInputProps {
  value: string[]
  onChange: (value: string[]) => void
  placeholder?: string
  className?: string
  maxTags?: number
}

export function ChipInput({ 
  value = [], 
  onChange, 
  placeholder = "Type and press Enter to add tags...", 
  className,
  maxTags = 10
}: ChipInputProps) {
  const [inputValue, setInputValue] = React.useState('')
  const inputRef = React.useRef<HTMLInputElement>(null)

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag()
    } else if (e.key === 'Backspace' && inputValue === '' && value.length > 0) {
      // Remove last tag when backspace is pressed on empty input
      removeTag(value.length - 1)
    }
  }

  const addTag = () => {
    const trimmedValue = inputValue.trim()
    if (trimmedValue && !value.includes(trimmedValue) && value.length < maxTags) {
      onChange([...value, trimmedValue])
      setInputValue('')
    }
  }

  const removeTag = (indexToRemove: number) => {
    onChange(value.filter((_, index) => index !== indexToRemove))
  }

  const handleContainerClick = () => {
    inputRef.current?.focus()
  }

  return (
    <div
      className={cn(
        "min-h-9 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs focus-within:border-ring focus-within:ring-ring/50 focus-within:ring-[3px] cursor-text",
        "dark:bg-input/30",
        className
      )}
      onClick={handleContainerClick}
    >
      <div className="flex flex-wrap gap-1 items-center">
        {value.map((tag, index) => (
          <Badge
            key={index}
            variant="secondary"
            className="h-6 px-2 py-0 text-xs gap-1"
          >
            {tag}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                removeTag(index)
              }}
              className="ml-1 rounded-full hover:bg-secondary-foreground/20"
            >
              <X className="h-3 w-3" />
            </button>
          </Badge>
        ))}
        <Input
          ref={inputRef}
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={addTag}
          placeholder={value.length === 0 ? placeholder : ''}
          className="flex-1 min-w-[120px] border-0 p-0 h-6 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          style={{ outline: 'none', boxShadow: 'none' }}
        />
      </div>
    </div>
  )
} 
