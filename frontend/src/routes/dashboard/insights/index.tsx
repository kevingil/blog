import { useState, useEffect } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Lightbulb, Tag, Calendar, Pin, Check, ExternalLink, Loader2, Search } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination';
import { useToast } from '@/hooks/use-toast';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { 
  listInsights, 
  listTopics, 
  markInsightAsRead, 
  toggleInsightPinned,
  searchInsights,
  type Insight,
  type InsightTopic,
} from '@/services/insights';

export const Route = createFileRoute('/dashboard/insights/')({
  component: InsightsPage,
});

function InsightsPage() {
  const [page, setPage] = useState(1);
  const [selectedTopicId, setSelectedTopicId] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();

  useEffect(() => {
    setPageTitle("Insights");
  }, [setPageTitle]);

  // Load topics
  const { data: topics = [] } = useQuery({
    queryKey: ['insight-topics'],
    queryFn: listTopics,
  });

  // Load insights
  const { data: insightsData, isLoading } = useQuery({
    queryKey: ['insights', page, selectedTopicId],
    queryFn: () => listInsights(page, 12, selectedTopicId === 'all' ? undefined : selectedTopicId),
  });

  // Search insights
  const { data: searchResults, isLoading: isSearchLoading } = useQuery({
    queryKey: ['insights-search', searchQuery],
    queryFn: () => searchInsights(searchQuery, 20),
    enabled: searchQuery.length > 2,
  });

  const insights = searchQuery.length > 2 ? searchResults || [] : insightsData?.insights || [];
  const total = insightsData?.total || 0;
  const totalPages = Math.ceil(total / 12);

  // Mutations
  const markReadMutation = useMutation({
    mutationFn: markInsightAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insights'] });
    },
  });

  const togglePinMutation = useMutation({
    mutationFn: toggleInsightPinned,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insights'] });
      toast({ title: 'Success', description: 'Insight pin status updated' });
    },
  });

  const handleMarkAsRead = async (insightId: string) => {
    markReadMutation.mutate(insightId);
  };

  const handleTogglePin = async (e: React.MouseEvent, insightId: string) => {
    e.stopPropagation();
    togglePinMutation.mutate(insightId);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto">
      {/* Header with filters */}
      <div className="flex flex-col md:flex-row gap-4 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search insights..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={selectedTopicId} onValueChange={(value) => {
          setSelectedTopicId(value);
          setPage(1);
        }}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Filter by topic" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Topics</SelectItem>
            {topics.map((topic) => (
              <SelectItem key={topic.id} value={topic.id}>
                {topic.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Link to="/dashboard/insights/topics">
          <Button variant="outline">
            <Tag className="w-4 h-4 mr-2" />
            Manage Topics
          </Button>
        </Link>
      </div>

      {isLoading || isSearchLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 animate-spin" />
          <span className="ml-2">Loading insights...</span>
        </div>
      ) : insights.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <Lightbulb className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg font-medium mb-2">No insights yet</p>
          <p className="text-sm">Insights are generated from your data sources. Add some data sources to get started.</p>
          <Link to="/dashboard/data-sources">
            <Button variant="outline" className="mt-4">
              Configure Data Sources
            </Button>
          </Link>
        </div>
      ) : (
        <>
          {/* Insights Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {insights.map((insight) => (
              <InsightCard
                key={insight.id}
                insight={insight}
                onMarkAsRead={handleMarkAsRead}
                onTogglePin={handleTogglePin}
                formatDate={formatDate}
              />
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && searchQuery.length <= 2 && (
            <div className="mt-6 flex justify-center">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => setPage(Math.max(1, page - 1))}
                      className={page === 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    const pageNum = i + 1;
                    return (
                      <PaginationItem key={pageNum}>
                        <PaginationLink
                          onClick={() => setPage(pageNum)}
                          isActive={page === pageNum}
                          className="cursor-pointer"
                        >
                          {pageNum}
                        </PaginationLink>
                      </PaginationItem>
                    );
                  })}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => setPage(Math.min(totalPages, page + 1))}
                      className={page === totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </>
      )}
    </section>
  );
}

interface InsightCardProps {
  insight: Insight;
  onMarkAsRead: (id: string) => void;
  onTogglePin: (e: React.MouseEvent, id: string) => void;
  formatDate: (date: string) => string;
}

function InsightCard({ insight, onMarkAsRead, onTogglePin, formatDate }: InsightCardProps) {
  return (
    <Link to={`/dashboard/insights/${insight.id}`} onClick={() => !insight.is_read && onMarkAsRead(insight.id)}>
      <Card className={`cursor-pointer transition-all hover:border-primary/50 ${!insight.is_read ? 'border-l-4 border-l-primary' : ''}`}>
        <CardHeader className="pb-2">
          <div className="flex items-start justify-between gap-2">
            <CardTitle className="text-sm font-medium line-clamp-2 flex-1">
              {insight.title}
            </CardTitle>
            <div className="flex items-center gap-1">
              {insight.is_pinned && (
                <Pin className="w-3 h-3 text-primary fill-primary" />
              )}
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={(e) => onTogglePin(e, insight.id)}
              >
                <Pin className={`w-3 h-3 ${insight.is_pinned ? 'fill-current' : ''}`} />
              </Button>
            </div>
          </div>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            {insight.topic_name && (
              <Badge variant="secondary" className="text-xs">
                <Tag className="w-3 h-3 mr-1" />
                {insight.topic_name}
              </Badge>
            )}
            {!insight.is_read && (
              <Badge variant="default" className="text-xs">New</Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <p className="text-sm text-muted-foreground line-clamp-3 mb-3">
            {insight.summary}
          </p>

          {insight.key_points && insight.key_points.length > 0 && (
            <div className="space-y-1 mb-3">
              {insight.key_points.slice(0, 2).map((point, i) => (
                <div key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                  <Check className="w-3 h-3 mt-0.5 flex-shrink-0 text-green-500" />
                  <span className="line-clamp-1">{point}</span>
                </div>
              ))}
              {insight.key_points.length > 2 && (
                <span className="text-xs text-muted-foreground">
                  +{insight.key_points.length - 2} more points
                </span>
              )}
            </div>
          )}

          <div className="flex items-center gap-2 text-xs text-muted-foreground border-t pt-2">
            <Calendar className="w-3 h-3" />
            <span>{formatDate(insight.generated_at)}</span>
            {insight.source_content_ids && insight.source_content_ids.length > 0 && (
              <>
                <span>•</span>
                <span>{insight.source_content_ids.length} sources</span>
              </>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
