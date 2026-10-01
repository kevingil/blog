import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { BookOpen, Eye } from 'lucide-react';
import { SourcesManagerContent } from '../SourcesManager';
import type { TextRange } from '@/lib/added-text';
import { TipTapEditor } from './TipTapEditor';

interface EditorTabsProps {
  content: string;
  onChange: (md: string) => void;
  highlights?: TextRange[];
  activeTab: string;
  onTabChange: (tab: string) => void;
  title?: string;
  authorName?: string;
  imageUrl?: string;
  tags?: string[];
  articleId?: string;
}

export function EditorTabs({
  content,
  onChange,
  highlights = [],
  activeTab,
  onTabChange,
  title,
  authorName,
  imageUrl,
  tags,
  articleId,
}: EditorTabsProps) {
  const tab = activeTab === 'resources' ? 'resources' : 'preview';

  return (
    <Tabs value={tab} onValueChange={onTabChange} className="flex h-full min-w-0 flex-col">
      <TabsList className="w-full shrink-0 justify-start rounded-none border-b bg-transparent px-1.5">
        <TabsTrigger value="preview" className="gap-1.5 data-[state=active]:bg-muted">
          <Eye className="h-3.5 w-3.5" />
          Preview
        </TabsTrigger>
        <TabsTrigger value="resources" className="gap-1.5 data-[state=active]:bg-muted">
          <BookOpen className="h-3.5 w-3.5" />
          Resources
        </TabsTrigger>
      </TabsList>

      <div className="relative min-h-0 flex-1">
        <div className={tab === 'preview' ? 'absolute inset-0' : 'absolute inset-0 hidden'}>
          <TipTapEditor
            content={content}
            onChange={onChange}
            highlights={highlights}
            title={title}
            authorName={authorName}
            imageUrl={imageUrl}
            tags={tags}
          />
        </div>
        {tab === 'resources' && (
          <div className="absolute inset-0 overflow-auto">
            <SourcesManagerContent articleId={articleId} />
          </div>
        )}
      </div>
    </Tabs>
  );
}
