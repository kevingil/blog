import { useRef, useEffect, useCallback } from 'react';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { MarkdownEditor } from './MarkdownEditor';
import { DiffView } from './DiffView';
import { MarkdownPreview } from './MarkdownPreview';
import { Code, Eye, ShieldCheck } from 'lucide-react';
import { EditorView } from '@codemirror/view';

interface EditorTabsProps {
  content: string;
  onChange: (md: string) => void;
  originalContent: string;
  diffing: boolean;
  activeTab: string;
  onTabChange: (tab: string) => void;
  onAccept: () => void;
  onReject: () => void;
  title?: string;
  authorName?: string;
  imageUrl?: string;
  tags?: string[];
}

/* ------------------------------------------------------------------ */
/* Line-based sync helpers (Edit ↔ Preview)                            */
/* ------------------------------------------------------------------ */

/** Get the 1-based line number at the top of a CodeMirror viewport */
function getFirstVisibleLine(view: EditorView): number {
  const block = view.lineBlockAtHeight(view.scrollDOM.scrollTop);
  return view.state.doc.lineAt(block.from).number;
}

/** Scroll CodeMirror so that `lineNumber` sits at the viewport top */
function scrollEditorToLine(view: EditorView, lineNumber: number): void {
  const clamped = Math.max(1, Math.min(lineNumber, view.state.doc.lines));
  const pos = view.state.doc.line(clamped).from;
  view.dispatch({ effects: EditorView.scrollIntoView(pos, { y: 'start' }) });
}

/**
 * Find the source line of the first visible `[data-source-line]` element
 * inside a Preview container.
 */
function getFirstVisibleSourceLine(container: HTMLElement): number | null {
  const scrollable = findScrollable(container);
  if (!scrollable) return null;
  const viewportTop = scrollable.getBoundingClientRect().top;

  const elements = container.querySelectorAll('[data-source-line]');
  for (const el of elements) {
    // First element whose bottom edge is at or below the viewport top
    if (el.getBoundingClientRect().bottom >= viewportTop) {
      return parseInt(el.getAttribute('data-source-line')!, 10);
    }
  }
  if (elements.length > 0) {
    return parseInt(elements[elements.length - 1].getAttribute('data-source-line')!, 10);
  }
  return null;
}

/**
 * Scroll the Preview panel so the element closest to `targetLine` is at the
 * top of the viewport.
 */
function scrollPreviewToLine(container: HTMLElement, targetLine: number): void {
  const elements = container.querySelectorAll('[data-source-line]');
  let bestEl: Element | null = null;
  let bestDelta = Infinity;

  for (const el of elements) {
    const line = parseInt(el.getAttribute('data-source-line')!, 10);
    const delta = Math.abs(line - targetLine);
    if (delta < bestDelta) {
      bestDelta = delta;
      bestEl = el;
    }
    if (line > targetLine) break;
  }

  if (bestEl) {
    bestEl.scrollIntoView({ block: 'start', behavior: 'instant' as ScrollBehavior });
  }
}

/* ------------------------------------------------------------------ */
/* Percentage-based fallback (Diff tab)                                */
/* ------------------------------------------------------------------ */

function findScrollable(container: HTMLElement | null): HTMLElement | null {
  if (!container) return null;
  const all = container.querySelectorAll('*');
  for (const el of all) {
    const htmlEl = el as HTMLElement;
    const style = getComputedStyle(htmlEl);
    if (
      (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
      htmlEl.scrollHeight > htmlEl.clientHeight + 1
    ) {
      return htmlEl;
    }
  }
  if (container.scrollHeight > container.clientHeight + 1) return container;
  return null;
}

/* ------------------------------------------------------------------ */
/* Component                                                           */
/* ------------------------------------------------------------------ */

export function EditorTabs({
  content,
  onChange,
  originalContent,
  diffing,
  activeTab,
  onTabChange,
  onAccept,
  onReject,
  title,
  authorName,
  imageUrl,
  tags,
}: EditorTabsProps) {
  const syncLineRef = useRef(1);
  const scrollFractionRef = useRef(0);
  const editorViewRef = useRef<EditorView | null>(null);
  const editRef = useRef<HTMLDivElement>(null);
  const previewRef = useRef<HTMLDivElement>(null);
  const diffRef = useRef<HTMLDivElement>(null);
  const prevTabRef = useRef(activeTab);

  /** Save scroll position from the outgoing tab, then switch */
  const handleTabChange = useCallback((newTab: string) => {
    if (activeTab === 'edit' && editorViewRef.current) {
      syncLineRef.current = getFirstVisibleLine(editorViewRef.current);
      const s = editorViewRef.current.scrollDOM;
      const max = s.scrollHeight - s.clientHeight;
      scrollFractionRef.current = max > 0 ? s.scrollTop / max : 0;
    } else if (activeTab === 'preview' && previewRef.current) {
      const line = getFirstVisibleSourceLine(previewRef.current);
      if (line != null) {
        syncLineRef.current = line;
        const total = content.split('\n').length;
        scrollFractionRef.current = total > 1 ? (line - 1) / (total - 1) : 0;
      }
    } else if (activeTab === 'diff' && diffRef.current) {
      const scrollable = findScrollable(diffRef.current);
      if (scrollable) {
        const max = scrollable.scrollHeight - scrollable.clientHeight;
        scrollFractionRef.current = max > 0 ? scrollable.scrollTop / max : 0;
        const total = content.split('\n').length;
        syncLineRef.current = Math.round(scrollFractionRef.current * Math.max(1, total - 1)) + 1;
      }
    }
    onTabChange(newTab);
  }, [activeTab, onTabChange, content]);

  /** Restore scroll position in the incoming tab */
  useEffect(() => {
    if (prevTabRef.current !== activeTab) {
      prevTabRef.current = activeTab;
      let cancelled = false;
      // Double rAF ensures layout is complete after conditional render
      requestAnimationFrame(() => {
        if (cancelled) return;
        requestAnimationFrame(() => {
          if (cancelled) return;
          if (activeTab === 'edit' && editorViewRef.current) {
            scrollEditorToLine(editorViewRef.current, syncLineRef.current);
          } else if (activeTab === 'preview' && previewRef.current) {
            scrollPreviewToLine(previewRef.current, syncLineRef.current);
          } else if (activeTab === 'diff' && diffRef.current) {
            const scrollable = findScrollable(diffRef.current);
            if (scrollable) {
              const max = scrollable.scrollHeight - scrollable.clientHeight;
              if (max > 0) scrollable.scrollTop = scrollFractionRef.current * max;
            }
          }
        });
      });
      return () => { cancelled = true; };
    }
  }, [activeTab]);

  return (
    <Tabs value={activeTab} onValueChange={handleTabChange} className="flex flex-col h-full min-w-0">
      <TabsList className="w-full justify-between rounded-none border-b bg-transparent px-2 shrink-0">
        <div className="flex">
          <TabsTrigger value="edit" className="gap-1.5 data-[state=active]:bg-muted">
            <Code className="h-3.5 w-3.5" />
            Edit
          </TabsTrigger>
          <TabsTrigger value="preview" className="gap-1.5 data-[state=active]:bg-muted">
            <Eye className="h-3.5 w-3.5" />
            Preview
          </TabsTrigger>
        </div>
        <div className="flex">
          <TabsTrigger value="diff" className="gap-1.5 data-[state=active]:bg-yellow-900/30 data-[state=active]:text-yellow-400 text-yellow-500/70">
            <ShieldCheck className="h-3.5 w-3.5" />
            Review
            {diffing && <span className="ml-1 h-2 w-2 rounded-full bg-yellow-400 animate-pulse" />}
          </TabsTrigger>
        </div>
      </TabsList>

      {/* Manual tab content -- avoids TabsContent flex issues with CodeMirror */}
      <div className="flex-1 min-h-0 relative">
        {/* Edit tab -- always mounted, hidden via CSS to preserve CodeMirror state */}
        <div
          ref={editRef}
          className="absolute inset-0"
          style={{ display: activeTab === 'edit' ? 'block' : 'none' }}
        >
          <MarkdownEditor content={content} onChange={onChange} editorViewRef={editorViewRef} />
        </div>

        {/* Diff/Review tab */}
        {activeTab === 'diff' && (
          <div ref={diffRef} className="absolute inset-0 overflow-auto">
            <DiffView
              oldValue={originalContent || content}
              newValue={content}
              onAccept={onAccept}
              onReject={onReject}
            />
          </div>
        )}

        {/* Preview tab */}
        {activeTab === 'preview' && (
          <div ref={previewRef} className="absolute inset-0 overflow-auto">
            <MarkdownPreview
              content={content}
              title={title}
              authorName={authorName}
              imageUrl={imageUrl}
              tags={tags}
            />
          </div>
        )}
      </div>
    </Tabs>
  );
}
