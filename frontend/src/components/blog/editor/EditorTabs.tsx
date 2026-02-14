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
  return (
    <Tabs value={activeTab} onValueChange={onTabChange} className="flex flex-col h-full min-w-0">
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
          className="absolute inset-0"
          style={{ display: activeTab === 'edit' ? 'block' : 'none' }}
        >
          <MarkdownEditor content={content} onChange={onChange} />
        </div>

        {/* Diff/Review tab */}
        {activeTab === 'diff' && (
          <div className="absolute inset-0 overflow-auto">
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
          <div className="absolute inset-0 overflow-auto">
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
