import { useState, type KeyboardEvent, type ReactNode } from "react"
import { ArrowUp, Link as LinkIcon, Loader2, Mic, Paperclip, Square, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  PromptInput,
  PromptInputAction,
  PromptInputActions,
  PromptInputTextarea,
} from "@/components/ui/prompt-input"
import { cn } from "@/lib/utils"

export interface AttachedSource {
  id: string
  url: string
  type: "url"
}

const EXAMPLES = [
  "Write a beginner's guide to React hooks",
  "Create an article about sustainable web design",
  "Explain the benefits of TypeScript for large projects",
  "Compare different state management solutions",
]

interface ChatComposerProps {
  variant: "landing" | "panel"
  value: string
  onValueChange: (value: string) => void
  onSubmit: (sources: AttachedSource[]) => void
  onVoice: (sources: AttachedSource[]) => void
  isLoading?: boolean
  disabled?: boolean
  placeholder?: string
  voiceOn?: boolean
  voiceDisabled?: boolean
  voiceStatus?: string | null
  accessory?: ReactNode
}

export function ChatComposer({
  variant,
  value,
  onValueChange,
  onSubmit,
  onVoice,
  isLoading = false,
  disabled = false,
  placeholder,
  voiceOn = false,
  voiceDisabled = false,
  voiceStatus,
  accessory,
}: ChatComposerProps) {
  const [sources, setSources] = useState<AttachedSource[]>([])
  const [showSourceInput, setShowSourceInput] = useState(false)
  const [sourceUrl, setSourceUrl] = useState("")
  const landing = variant === "landing"
  const busy = disabled || isLoading

  const addSource = () => {
    const url = sourceUrl.trim()
    if (!url) return
    setSources((current) => [...current, { id: Date.now().toString(), url, type: "url" }])
    setSourceUrl("")
    setShowSourceInput(false)
  }

  const submit = () => {
    if (busy) return
    onSubmit(landing ? sources : [])
  }

  return (
    <div className={cn(landing ? "space-y-0" : "space-y-1.5")}>
      {!landing && (
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <VoiceButton
              on={voiceOn}
              disabled={voiceDisabled}
              compact
              onClick={() => onVoice([])}
            />
            {voiceOn && (
              <span className="truncate text-[11px] text-muted-foreground">
                {voiceStatus}
              </span>
            )}
          </div>
          {accessory}
        </div>
      )}

      <PromptInput
        value={value}
        onValueChange={onValueChange}
        onSubmit={submit}
        isLoading={isLoading}
        disabled={disabled}
        maxHeight={landing ? 320 : 240}
        className={cn(
          landing && "rounded-2xl border bg-card p-0 shadow-sm focus-within:border-ring/40 focus-within:ring-2 focus-within:ring-ring/30",
        )}
      >
        <PromptInputTextarea
          placeholder={placeholder}
          className={cn(landing && "min-h-[100px] px-6 pt-6 text-base")}
        />

        {landing && sources.length > 0 && (
          <div className="flex flex-wrap gap-2 px-6 pb-3">
            {sources.map((source) => (
              <div
                key={source.id}
                className="group inline-flex items-center gap-2 rounded-lg border bg-muted px-3 py-1.5 text-sm"
              >
                <LinkIcon className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="max-w-[200px] truncate">{source.url}</span>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation()
                    setSources((current) => current.filter((item) => item.id !== source.id))
                  }}
                  className="opacity-0 transition-opacity group-hover:opacity-100"
                  disabled={busy}
                >
                  <X className="h-3.5 w-3.5 text-muted-foreground" />
                </button>
              </div>
            ))}
          </div>
        )}

        {landing && showSourceInput && (
          <div
            className="px-6 pb-3"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex gap-2">
              <input
                type="url"
                value={sourceUrl}
                onChange={(event) => setSourceUrl(event.target.value)}
                placeholder="https://example.com/article"
                className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring/30"
                onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => {
                  if (event.key === "Enter") {
                    event.preventDefault()
                    event.stopPropagation()
                    addSource()
                  }
                  if (event.key === "Escape") {
                    setShowSourceInput(false)
                    setSourceUrl("")
                  }
                }}
                autoFocus
              />
              <Button type="button" size="sm" onClick={addSource} disabled={!sourceUrl.trim()}>
                Add
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                onClick={() => {
                  setShowSourceInput(false)
                  setSourceUrl("")
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {landing ? (
          <div className="flex items-center justify-between border-t px-4 pb-4 pt-2">
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={(event) => {
                  event.stopPropagation()
                  setShowSourceInput((open) => !open)
                }}
                disabled={busy}
                className="text-muted-foreground hover:text-foreground"
              >
                <Paperclip className="mr-1.5 h-4 w-4" />
                Add source
              </Button>
              {value.length > 0 && (
                <span className="hidden text-xs text-muted-foreground sm:inline">
                  {value.length} characters
                </span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <span className="hidden text-xs text-muted-foreground sm:inline">Enter</span>
              <VoiceButton
                on={voiceOn}
                disabled={voiceDisabled || busy}
                onClick={() => onVoice(sources)}
              />
              <Button
                type="button"
                onClick={(event) => {
                  event.stopPropagation()
                  submit()
                }}
                disabled={busy || !value.trim()}
                className="shadow-sm"
              >
                {isLoading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <ArrowUp className="mr-2 h-4 w-4" />
                )}
                Send
              </Button>
            </div>
          </div>
        ) : (
          <PromptInputActions className="justify-end pt-2">
            <PromptInputAction tooltip={isLoading ? "Stop generation" : "Send"}>
              <Button
                type="button"
                variant="default"
                size="icon"
                className="h-8 w-8 rounded-full"
                disabled={!isLoading && !value.trim()}
                onClick={(event) => {
                  event.stopPropagation()
                  submit()
                }}
              >
                {isLoading ? (
                  <Square className="size-5 fill-current" />
                ) : (
                  <ArrowUp className="size-5" />
                )}
              </Button>
            </PromptInputAction>
          </PromptInputActions>
        )}
      </PromptInput>
    </div>
  )
}

function VoiceButton({
  on,
  disabled,
  compact = false,
  onClick,
}: {
  on: boolean
  disabled?: boolean
  compact?: boolean
  onClick: () => void
}) {
  return (
    <Button
      type="button"
      size={compact ? "sm" : "icon"}
      variant={on ? "default" : "outline"}
      className={cn(
        compact ? "h-7 w-7 px-0" : "h-9 w-9",
        on && "ring-2 ring-primary ring-offset-2 ring-offset-background",
      )}
      aria-pressed={on}
      aria-label={on ? "Turn voice off" : "Turn voice on"}
      title={on ? "Voice on" : "Voice"}
      disabled={disabled}
      onClick={(event) => {
        event.stopPropagation()
        onClick()
      }}
    >
      <Mic className={compact ? "h-3.5 w-3.5" : "h-4 w-4"} />
    </Button>
  )
}

export function WritingSuggestions({ onPick }: { onPick: (prompt: string) => void }) {
  return (
    <div className="mt-6 space-y-3">
      <p className="text-center text-sm text-muted-foreground">Try an example:</p>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {EXAMPLES.map((example) => (
          <button
            key={example}
            type="button"
            onClick={() => onPick(example)}
            className="group rounded-xl border bg-card p-4 text-left text-sm transition-all duration-200 hover:border-accent-foreground/20 hover:bg-accent"
          >
            <span className="text-muted-foreground transition-colors group-hover:text-foreground">
              {example}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}
