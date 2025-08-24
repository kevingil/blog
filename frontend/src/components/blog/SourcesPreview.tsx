import { useState, useEffect } from 'react';
import { BookOpen, Globe, FileText, Plus } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { getArticleSources } from '@/services/sources';
import type { ArticleSource } from '@/services/types';

interface SourcesPreviewProps {
  articleId: string | undefined;
  onOpenDrawer: () => void;
  disabled?: boolean;
  refreshTrigger?: number; // Optional prop to trigger refresh
}

export function SourcesPreview({ articleId, onOpenDrawer, disabled, refreshTrigger }: SourcesPreviewProps) {
  const [sources, setSources] = useState<ArticleSource[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (articleId) {
      loadSources();
    }
  }, [articleId, refreshTrigger]);

  const loadSources = async () => {
    if (!articleId) return;
    
    setIsLoading(true);
    try {
      const sourcesData = await getArticleSources(articleId);
      setSources(sourcesData);
    } catch (error) {
      console.error('Error loading sources:', error);
      setSources([]);
    } finally {
      setIsLoading(false);
    }
  };

  const truncateTitle = (title: string, maxLength: number = 25) => {
    return title.length > maxLength ? title.substring(0, maxLength) + '...' : title;
  };

  return (
    <div className="flex flex-col gap-2">
      <Card 
        className="w-full h-32 p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors overflow-hidden"
        onClick={!disabled ? onOpenDrawer : undefined}
      >
        {disabled ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <BookOpen className="w-6 h-6 mx-auto mb-1 opacity-50" />
              <span className="text-xs">Save article first</span>
            </div>
          </div>
        ) : isLoading ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin mx-auto mb-1" />
              <span className="text-xs">Loading...</span>
            </div>
          </div>
        ) : sources.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <div className="text-center">
              <Plus className="w-6 h-6 mx-auto mb-1 opacity-50" />
              <span className="text-xs">Add sources</span>
            </div>
          </div>
        ) : (
          <div className="h-full flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-1">
                <BookOpen className="w-3 h-3 text-indigo-500" />
                <span className="text-xs font-medium text-gray-700 dark:text-gray-300">
                  {sources.length} source{sources.length !== 1 ? 's' : ''}
                </span>
              </div>
              <Button variant="ghost" size="sm" className="h-5 px-1 text-xs">
                View all
              </Button>
            </div>
            
            {/* Sources List */}
            <div className="flex-1 space-y-1 overflow-y-auto">
              {sources.slice(0, 3).map((source) => (
                <div 
                  key={source.id} 
                  className="flex items-center gap-2 p-1 rounded text-xs bg-gray-50 dark:bg-gray-800/50"
                >
                  {source.source_type === 'web' ? (
                    <Globe className="w-3 h-3 text-blue-500 flex-shrink-0" />
                  ) : (
                    <FileText className="w-3 h-3 text-gray-500 flex-shrink-0" />
                  )}
                  <span className="flex-1 truncate text-gray-700 dark:text-gray-300">
                    {truncateTitle(source.title)}
                  </span>
                </div>
              ))}
              
              {sources.length > 3 && (
                <div className="text-xs text-muted-foreground text-center pt-1">
                  +{sources.length - 3} more
                </div>
              )}
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
