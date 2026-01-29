import { useEffect } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Tag, Calendar, Pin, ExternalLink, Loader2, Check, FileText } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { 
  getInsight, 
  markInsightAsRead, 
  toggleInsightPinned,
} from '@/services/insights';

export const Route = createFileRoute('/dashboard/insights/$insightId')({
  component: InsightDetailPage,
});

function InsightDetailPage() {
  const { insightId } = Route.useParams();
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();

  const { data: insight, isLoading, error } = useQuery({
    queryKey: ['insight', insightId],
    queryFn: () => getInsight(insightId),
  });

  useEffect(() => {
    if (insight) {
      setPageTitle(insight.title);
      // Mark as read on view
      if (!insight.is_read) {
        markInsightAsRead(insightId);
        queryClient.invalidateQueries({ queryKey: ['insights'] });
      }
    }
  }, [insight, insightId, setPageTitle, queryClient]);

  const togglePinMutation = useMutation({
    mutationFn: () => toggleInsightPinned(insightId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insight', insightId] });
      queryClient.invalidateQueries({ queryKey: ['insights'] });
      toast({ title: 'Success', description: 'Pin status updated' });
    },
  });

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-6 h-6 animate-spin" />
        <span className="ml-2">Loading insight...</span>
      </div>
    );
  }

  if (error || !insight) {
    return (
      <div className="text-center py-12">
        <p className="text-destructive">Failed to load insight</p>
        <Link to="/dashboard/insights">
          <Button variant="outline" className="mt-4">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Insights
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto max-w-4xl mx-auto">
      {/* Back link */}
      <Link to="/dashboard/insights" className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground mb-4">
        <ArrowLeft className="w-4 h-4 mr-1" />
        Back to Insights
      </Link>

      {/* Header */}
      <div className="mb-6">
        <div className="flex items-start justify-between gap-4">
          <h1 className="text-2xl font-bold">{insight.title}</h1>
          <Button
            variant={insight.is_pinned ? 'default' : 'outline'}
            size="sm"
            onClick={() => togglePinMutation.mutate()}
          >
            <Pin className={`w-4 h-4 mr-2 ${insight.is_pinned ? 'fill-current' : ''}`} />
            {insight.is_pinned ? 'Pinned' : 'Pin'}
          </Button>
        </div>
        
        <div className="flex items-center gap-3 mt-3 flex-wrap">
          {insight.topic && (
            <Badge variant="secondary">
              <Tag className="w-3 h-3 mr-1" />
              {insight.topic.name}
            </Badge>
          )}
          <span className="text-sm text-muted-foreground flex items-center">
            <Calendar className="w-3 h-3 mr-1" />
            {formatDate(insight.generated_at)}
          </span>
          {insight.source_content_ids && insight.source_content_ids.length > 0 && (
            <span className="text-sm text-muted-foreground">
              {insight.source_content_ids.length} sources
            </span>
          )}
        </div>
      </div>

      {/* Summary */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-lg">Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground leading-relaxed">{insight.summary}</p>
        </CardContent>
      </Card>

      {/* Key Points */}
      {insight.key_points && insight.key_points.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-lg">Key Points</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {insight.key_points.map((point, i) => (
                <li key={i} className="flex items-start gap-3">
                  <Check className="w-5 h-5 mt-0.5 flex-shrink-0 text-green-500" />
                  <span>{point}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Full Content */}
      {insight.content && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-lg">Full Analysis</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="prose prose-sm dark:prose-invert max-w-none">
              {insight.content.split('\n').map((paragraph, i) => (
                paragraph.trim() && <p key={i}>{paragraph}</p>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Source Content */}
      {insight.source_contents && insight.source_contents.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Source Content</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {insight.source_contents.map((content) => (
                <div key={content.id} className="border rounded-lg p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <h4 className="font-medium">{content.title || 'Untitled'}</h4>
                      <a
                        href={content.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-primary hover:underline flex items-center gap-1 mt-1"
                      >
                        <ExternalLink className="w-3 h-3" />
                        {content.url}
                      </a>
                    </div>
                    {content.published_at && (
                      <span className="text-xs text-muted-foreground">
                        {new Date(content.published_at).toLocaleDateString()}
                      </span>
                    )}
                  </div>
                  {content.summary && (
                    <p className="text-sm text-muted-foreground mt-2">{content.summary}</p>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </section>
  );
}
