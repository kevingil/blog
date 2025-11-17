import { useState, useRef, KeyboardEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Sparkles, Paperclip, X, Link as LinkIcon, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface AttachedSource {
  id: string;
  url: string;
  type: 'url';
}

interface AIChatLandingProps {
  onGenerate: (prompt: string, sources: AttachedSource[]) => Promise<void>;
  isGenerating?: boolean;
}

export function AIChatLanding({ onGenerate, isGenerating = false }: AIChatLandingProps) {
  const [prompt, setPrompt] = useState('');
  const [sources, setSources] = useState<AttachedSource[]>([]);
  const [showSourceInput, setShowSourceInput] = useState(false);
  const [sourceUrl, setSourceUrl] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = async () => {
    if (!prompt.trim() || isGenerating) return;
    await onGenerate(prompt.trim(), sources);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const addSource = () => {
    if (!sourceUrl.trim()) return;
    
    const newSource: AttachedSource = {
      id: Date.now().toString(),
      url: sourceUrl.trim(),
      type: 'url'
    };
    
    setSources([...sources, newSource]);
    setSourceUrl('');
    setShowSourceInput(false);
  };

  const removeSource = (id: string) => {
    setSources(sources.filter(s => s.id !== id));
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-4rem)] w-full px-4 py-8">
      <div className="w-full max-w-3xl space-y-6">
        {/* Header with animated gradient */}
        <div className="text-center space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-700">
          <div className="flex items-center justify-center gap-2">
          <div className="inline-flex items-center justify-center p-3 rounded-2xl bg-gradient-to-br from-violet-500/10 to-indigo-500/10 mb-2">
            <Sparkles className="h-8 w-8 text-violet-600 dark:text-violet-400" />
          </div>
          <h1 className="text-3xl md:text-4xl font-semibold tracking-tight">
            What are we writing today?
          </h1>
          </div>
          {/* <p className="text-muted-foreground text-base md:text-lg">
            Describe your article idea and AI will help you write it
          </p> */}
        </div>

        {/* Main input area */}
        <div className="animate-in fade-in slide-in-from-bottom-4 duration-700 delay-150">
          <div className={cn(
            "relative rounded-2xl border bg-card shadow-sm transition-all duration-200",
            "focus-within:ring-2 focus-within:ring-violet-500/20 focus-within:border-violet-500/50",
            isGenerating && "opacity-60 pointer-events-none"
          )}>
            {/* Textarea */}
            <Textarea
              ref={textareaRef}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Write an article about..."
              className="min-h-[100px] resize-none border-0 focus-visible:ring-0 text-base p-6 bg-transparent"
              disabled={isGenerating}
            />

            {/* Attached sources */}
            {sources.length > 0 && (
              <div className="px-6 pb-3 flex flex-wrap gap-2">
                {sources.map((source) => (
                  <div
                    key={source.id}
                    className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-violet-50 dark:bg-violet-950/30 border border-violet-200 dark:border-violet-800 text-sm group"
                  >
                    <LinkIcon className="h-3.5 w-3.5 text-violet-600 dark:text-violet-400" />
                    <span className="text-violet-700 dark:text-violet-300 max-w-[200px] truncate">
                      {source.url}
                    </span>
                    <button
                      onClick={() => removeSource(source.id)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity"
                      disabled={isGenerating}
                    >
                      <X className="h-3.5 w-3.5 text-violet-600 dark:text-violet-400" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Source input */}
            {showSourceInput && (
              <div className="px-6 pb-3 animate-in fade-in slide-in-from-top-2 duration-200">
                <div className="flex gap-2">
                  <input
                    type="url"
                    value={sourceUrl}
                    onChange={(e) => setSourceUrl(e.target.value)}
                    placeholder="https://example.com/article"
                    className="flex-1 px-3 py-2 text-sm rounded-lg border bg-background focus:outline-none focus:ring-2 focus:ring-violet-500/20"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        addSource();
                      }
                      if (e.key === 'Escape') {
                        setShowSourceInput(false);
                        setSourceUrl('');
                      }
                    }}
                    autoFocus
                  />
                  <Button
                    size="sm"
                    onClick={addSource}
                    disabled={!sourceUrl.trim()}
                  >
                    Add
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      setShowSourceInput(false);
                      setSourceUrl('');
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            )}

            {/* Bottom toolbar */}
            <div className="flex items-center justify-between px-4 pb-4 pt-2 border-t">
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowSourceInput(!showSourceInput)}
                  disabled={isGenerating}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <Paperclip className="h-4 w-4 mr-1.5" />
                  Add source
                </Button>
                
                <span className="text-xs text-muted-foreground hidden sm:inline">
                  {prompt.length > 0 && `${prompt.length} characters`}
                </span>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground hidden sm:inline">
                  Cmd/Ctrl + Enter
                </span>
                <Button
                  onClick={handleSubmit}
                  disabled={!prompt.trim() || isGenerating}
                  className="bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-700 hover:to-indigo-700 text-white shadow-sm"
                >
                  {isGenerating ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Generating...
                    </>
                  ) : (
                    <>
                      <Sparkles className="h-4 w-4 mr-2" />
                      Generate
                    </>
                  )}
                </Button>
              </div>
            </div>
          </div>

          {/* Helper text */}
          {/* <div className="flex items-center justify-center gap-6 mt-6 text-xs text-muted-foreground">
            <span className="flex items-center gap-1.5">
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500" />
              AI-powered writing
            </span>
            <span className="flex items-center gap-1.5">
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-500" />
              Attach sources
            </span>
            <span className="flex items-center gap-1.5">
              <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-500" />
              Edit after generation
            </span>
          </div> */}
        </div>

        {/* Example prompts - optional, can be animated in */}
        {prompt.length === 0 && !isGenerating && (
          <div className="space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-700 delay-300">
            <p className="text-sm text-muted-foreground text-center">Try an example:</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {[
                "Write a beginner's guide to React hooks",
                "Create an article about sustainable web design",
                "Explain the benefits of TypeScript for large projects",
                "Compare different state management solutions"
              ].map((example, i) => (
                <button
                  key={i}
                  onClick={() => setPrompt(example)}
                  className="text-left p-4 rounded-xl border bg-card hover:bg-accent hover:border-accent-foreground/20 transition-all duration-200 text-sm group"
                >
                  <span className="text-muted-foreground group-hover:text-foreground transition-colors">
                    {example}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

