import { useState, useEffect } from 'react';
import { BookOpen, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
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

  return (
    <Button 
      variant="outline" 
      size="sm" 
      onClick={onOpenDrawer} 
      disabled={disabled || isLoading}
    >
      {isLoading ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <BookOpen className="h-4 w-4" />
      )}
      Sources
      {sources.length > 0 && (
        <Badge variant="secondary" className="ml-1">
          {sources.length}
        </Badge>
      )}
    </Button>
  );
}
