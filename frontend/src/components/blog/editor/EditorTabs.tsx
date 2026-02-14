import { useRef, useEffect, useCallback } from 'react';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { MarkdownEditor } from './MarkdownEditor';
import { DiffView } from './DiffView';
import { MarkdownPreview } from './MarkdownPreview';
import { Code, Eye, ShieldCheck } from 'lucide-react';

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

/**
 * Find the primary scrollable element inside a container.
 * CodeMirror uses `.cm-scroller`; other panels use the first descendant
 * with overflow-y auto/scroll that has overflowing content.
 */
function findScrollable(container: HTMLElement | null): HTMLElement | null {
  if (!container) return null;
  const cmScroller = container.querySelector('.cm-scroller');
  if (cmScroller) return cmScroller as HTMLElement;
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
  const scrollFractionRef = useRef(0);
  const editRef = useRef<HTMLDivElement>(null);
  const previewRef = useRef<HTMLDivElement>(null);
  const diffRef = useRef<HTMLDivElement>(null);
  const prevTabRef = useRef(activeTab);

  const getRefForTab = useCallback((tab: string) => {
    if (tab === 'edit') return editRef;
    if (tab === 'preview') return previewRef;
    return diffRef;
  }, []);

  /** Intercept tab changes to save scroll position before switching */
  const handleTabChange = useCallback((newTab: string) => {
    const scrollable = findScrollable(getRefForTab(activeTab).current);
    if (scrollable) {
      const maxScroll = scrollable.scrollHeight - scrollable.clientHeight;
      scrollFractionRef.current = maxScroll > 0 ? scrollable.scrollTop / maxScroll : 0;
    }
    onTabChange(newTab);
  }, [activeTab, onTabChange, getRefForTab]);

  /** Restore scroll position when the active tab changes */
  useEffect(() => {
    if (prevTabRef.current !== activeTab) {
      prevTabRef.current = activeTab;
      let cancelled = false;
      // Double rAF ensures layout is complete after conditional render
      requestAnimationFrame(() => {
        if (cancelled) return;
        requestAnimationFrame(() => {
          if (cancelled) return;
          const scrollable = findScrollable(getRefForTab(activeTab).current);
          if (scrollable) {
            const maxScroll = scrollable.scrollHeight - scrollable.clientHeight;
            if (maxScroll > 0) {
              scrollable.scrollTop = scrollFractionRef.current * maxScroll;
            }
          }
        });
      });
      return () => { cancelled = true; };
    }
  }, [activeTab, getRefForTab]);

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
          <MarkdownEditor content={content} onChange={onChange} />
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
