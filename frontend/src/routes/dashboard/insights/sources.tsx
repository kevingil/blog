import { useCallback, useEffect, useRef, useState } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertCircle,
  ArrowLeft,
  ArrowUp,
  CheckCircle,
  CheckCircle2,
  ChevronDown,
  Clock,
  ExternalLink,
  Globe,
  Loader2,
  Pencil,
  Play,
  Plus,
  RefreshCw,
  Search,
  Sparkles,
  Square,
  Trash2,
  XCircle,
  Zap,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { useToast } from '@/hooks/use-toast';
import { ApiError } from '@/services/authenticatedFetch';
import { VITE_WS_URL } from '@/services/constants';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import {
  createDataSource,
  deleteDataSource,
  listDataSources,
  recommendDataSources,
  triggerCrawl,
  updateDataSource,
  type CreateDataSourceRequest,
  type DataSource,
  type DataSourceRecommendation,
  type RecommendDataSourcesResponse,
  type UpdateDataSourceRequest,
} from '@/services/dataSources';
import {
  getWorkerDescription,
  getWorkerDisplayName,
  getWorkersStatus,
  runWorker,
  stopWorker,
  type WorkerState,
  type WorkerStatus,
} from '@/services/workers';

export const Route = createFileRoute('/dashboard/insights/sources')({
  component: InsightSourcesPage,
});

const workerIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  crawl: RefreshCw,
  insight: Sparkles,
  discovery: Search,
};

type RecommendationSaveState = 'idle' | 'adding' | 'added' | 'error';

interface RecommendationStatus {
  state: RecommendationSaveState;
  message?: string;
}

const recommendationKey = (recommendation: DataSourceRecommendation) => recommendation.url;

const getRecommendationTypeVariant = (sourceType: string): 'default' | 'secondary' | 'outline' => {
  switch (sourceType) {
    case 'news':
      return 'default';
    case 'newsletter':
      return 'secondary';
    default:
      return 'outline';
  }
};

const getWorkerStateColor = (state: WorkerState): string => {
  switch (state) {
    case 'running':
      return 'text-blue-500';
    case 'completed':
      return 'text-green-500';
    case 'failed':
      return 'text-destructive';
    default:
      return 'text-muted-foreground';
  }
};

const getWorkerStateBgColor = (state: WorkerState): string => {
  switch (state) {
    case 'running':
      return 'bg-blue-500/10';
    case 'completed':
      return 'bg-green-500/10';
    case 'failed':
      return 'bg-destructive/10';
    default:
      return 'bg-muted';
  }
};

function InsightSourcesPage() {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<DataSource | null>(null);
  const [formData, setFormData] = useState<CreateDataSourceRequest>({
    name: '',
    url: '',
    source_type: 'blog',
    crawl_frequency: 'daily',
    is_enabled: true,
  });
  const [queryInput, setQueryInput] = useState('');
  const [activeSearchQuery, setActiveSearchQuery] = useState('');
  const [selectedRecommendations, setSelectedRecommendations] = useState<Record<string, boolean>>({});
  const [recommendationStatuses, setRecommendationStatuses] = useState<Record<string, RecommendationStatus>>({});
  const [isAddingRecommendations, setIsAddingRecommendations] = useState(false);
  const [workerStatuses, setWorkerStatuses] = useState<Record<string, WorkerStatus>>({});
  const [runningWorker, setRunningWorker] = useState<string | null>(null);
  const [workerControlsOpen, setWorkerControlsOpen] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();

  useEffect(() => {
    setPageTitle('Insights Sources');
  }, [setPageTitle]);

  useEffect(() => {
    getWorkersStatus()
      .then((response) => {
        const statuses: Record<string, WorkerStatus> = {};
        response.workers.forEach((worker) => {
          statuses[worker.name] = worker;
        });
        setWorkerStatuses(statuses);
      })
      .catch(console.error);
  }, []);

  useEffect(() => {
    const wsURL = VITE_WS_URL || `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/websocket`;
    const ws = new WebSocket(wsURL);
    wsRef.current = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({
        action: 'subscribe',
        channel: 'worker-status',
      }));
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type !== 'worker-status') {
          return;
        }

        setWorkerStatuses((prev) => ({
          ...prev,
          [data.worker_name]: data.status,
        }));

        if (data.status.state === 'completed') {
          toast({
            title: `${getWorkerDisplayName(data.worker_name)} completed`,
            description: data.status.message || 'Worker completed successfully.',
          });
          if (data.worker_name === 'crawl') {
            queryClient.invalidateQueries({ queryKey: ['data-sources'] });
          }
        } else if (data.status.state === 'failed') {
          toast({
            title: `${getWorkerDisplayName(data.worker_name)} failed`,
            description: data.status.error || 'Worker failed.',
            variant: 'destructive',
          });
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          action: 'unsubscribe',
          channel: 'worker-status',
        }));
      }
      ws.close();
    };
  }, [toast, queryClient]);

  const { data, isLoading } = useQuery({
    queryKey: ['data-sources'],
    queryFn: () => listDataSources(),
  });

  const recommendMutation = useMutation({
    mutationFn: recommendDataSources,
    onSuccess: (response) => {
      const nextSelected: Record<string, boolean> = {};
      response.recommendations.forEach((recommendation) => {
        nextSelected[recommendationKey(recommendation)] = false;
      });
      setSelectedRecommendations(nextSelected);
      setRecommendationStatuses({});
      if (response.recommendations.length === 0) {
        toast({
          title: 'No source recommendations found',
          description: 'Try a narrower topic or add a source manually.',
        });
      }
    },
    onError: (error) => {
      toast({
        title: 'Source search failed',
        description: error instanceof Error ? error.message : 'Unable to load source recommendations.',
        variant: 'destructive',
      });
    },
  });

  const createMutation = useMutation({
    mutationFn: createDataSource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Source added', description: 'Insight source created successfully.' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to create data source.', variant: 'destructive' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateDataSourceRequest }) => updateDataSource(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Source updated', description: 'Insight source updated successfully.' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to update data source.', variant: 'destructive' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteDataSource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Source deleted', description: 'Insight source removed.' });
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to delete data source.', variant: 'destructive' });
    },
  });

  const crawlMutation = useMutation({
    mutationFn: triggerCrawl,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Crawl queued', description: 'Source crawl triggered successfully.' });
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to trigger crawl.', variant: 'destructive' });
    },
  });

  const dataSources: DataSource[] = Array.isArray(data) ? data : (data as any)?.data_sources || [];
  const recommendations = recommendMutation.data?.recommendations || [];
  const isSearchMode = activeSearchQuery.length > 0;
  const selectedRecommendationCount = recommendations.filter((recommendation) => {
    const key = recommendationKey(recommendation);
    return selectedRecommendations[key] && recommendationStatuses[key]?.state !== 'added';
  }).length;

  const handleOpenCreate = useCallback(() => {
    setEditingSource(null);
    setFormData({
      name: '',
      url: '',
      source_type: 'blog',
      crawl_frequency: 'daily',
      is_enabled: true,
    });
    setIsDialogOpen(true);
  }, []);

  const handleOpenEdit = useCallback((source: DataSource) => {
    setEditingSource(source);
    setFormData({
      name: source.name,
      url: source.url,
      feed_url: source.feed_url,
      source_type: source.source_type,
      crawl_frequency: source.crawl_frequency,
      is_enabled: source.is_enabled,
    });
    setIsDialogOpen(true);
  }, []);

  const handleCloseDialog = useCallback(() => {
    setIsDialogOpen(false);
    setEditingSource(null);
  }, []);

  const clearSearch = useCallback(() => {
    setQueryInput('');
    setActiveSearchQuery('');
    setSelectedRecommendations({});
    setRecommendationStatuses({});
    recommendMutation.reset();
  }, [recommendMutation]);

  const handleSearch = useCallback(() => {
    const trimmedQuery = queryInput.trim();
    if (!trimmedQuery || recommendMutation.isPending) {
      return;
    }

    setActiveSearchQuery(trimmedQuery);
    recommendMutation.mutate({
      query: trimmedQuery,
      limit: 8,
    });
  }, [queryInput, recommendMutation]);

  const handleToggleRecommendation = useCallback((key: string, checked: boolean) => {
    setSelectedRecommendations((prev) => ({
      ...prev,
      [key]: checked,
    }));
  }, []);

  const handleToggleAllRecommendations = useCallback((checked: boolean) => {
    setSelectedRecommendations((prev) => {
      const next = { ...prev };
      recommendations.forEach((recommendation) => {
        const key = recommendationKey(recommendation);
        if (recommendationStatuses[key]?.state !== 'added') {
          next[key] = checked;
        }
      });
      return next;
    });
  }, [recommendations, recommendationStatuses]);

  const saveRecommendations = useCallback(async (items: DataSourceRecommendation[]) => {
    if (items.length === 0 || isAddingRecommendations) {
      return;
    }

    setIsAddingRecommendations(true);
    setRecommendationStatuses((prev) => {
      const next = { ...prev };
      items.forEach((recommendation) => {
        next[recommendationKey(recommendation)] = { state: 'adding' };
      });
      return next;
    });

    const results = await Promise.allSettled(items.map(async (recommendation) => {
      await createDataSource({
        name: recommendation.name,
        url: recommendation.url,
        source_type: recommendation.source_type,
        crawl_frequency: 'daily',
        is_enabled: true,
      });
      return recommendation;
    }));

    let successCount = 0;
    let errorCount = 0;

    setRecommendationStatuses((prev) => {
      const next = { ...prev };

      results.forEach((result, index) => {
        const recommendation = items[index];
        const key = recommendationKey(recommendation);

        if (result.status === 'fulfilled') {
          successCount += 1;
          next[key] = { state: 'added', message: 'Added' };
          return;
        }

        errorCount += 1;
        const message = result.reason instanceof ApiError
          ? result.reason.message
          : result.reason instanceof Error
            ? result.reason.message
            : 'Failed to add source';
        next[key] = { state: 'error', message };
      });

      return next;
    });

    setSelectedRecommendations((prev) => {
      const next = { ...prev };
      items.forEach((recommendation) => {
        next[recommendationKey(recommendation)] = false;
      });
      return next;
    });

    if (successCount > 0) {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({
        title: successCount === 1 ? 'Source added' : `${successCount} sources added`,
        description: errorCount > 0
          ? `${errorCount} recommendation${errorCount === 1 ? '' : 's'} could not be added.`
          : 'Recommended sources are now in your sources table.',
      });
    } else {
      toast({
        title: 'No sources were added',
        description: 'Review the per-source errors and try again.',
        variant: 'destructive',
      });
    }

    setIsAddingRecommendations(false);
  }, [isAddingRecommendations, queryClient, toast]);

  const handleAddRecommendation = useCallback((recommendation: DataSourceRecommendation) => {
    saveRecommendations([recommendation]);
  }, [saveRecommendations]);

  const handleAddSelectedRecommendations = useCallback(() => {
    const items = recommendations.filter((recommendation) => {
      const key = recommendationKey(recommendation);
      return selectedRecommendations[key] && recommendationStatuses[key]?.state !== 'added';
    });
    saveRecommendations(items);
  }, [recommendations, selectedRecommendations, recommendationStatuses, saveRecommendations]);

  const handleSubmit = useCallback(() => {
    if (editingSource) {
      updateMutation.mutate({ id: editingSource.id, payload: formData });
    } else {
      createMutation.mutate(formData);
    }
  }, [createMutation, editingSource, formData, updateMutation]);

  const handleRunWorker = useCallback(async (name: string) => {
    setRunningWorker(name);
    try {
      await runWorker(name);
      toast({
        title: 'Worker started',
        description: `${getWorkerDisplayName(name)} is now running.`,
      });
    } catch (error) {
      toast({
        title: 'Failed to start worker',
        description: error instanceof Error ? error.message : 'Unknown error.',
        variant: 'destructive',
      });
    } finally {
      setRunningWorker(null);
    }
  }, [toast]);

  const handleStopWorker = useCallback(async (name: string) => {
    setRunningWorker(name);
    try {
      await stopWorker(name);
      toast({
        title: 'Worker stopped',
        description: `${getWorkerDisplayName(name)} has been stopped.`,
      });
    } catch (error) {
      toast({
        title: 'Failed to stop worker',
        description: error instanceof Error ? error.message : 'Unknown error.',
        variant: 'destructive',
      });
    } finally {
      setRunningWorker(null);
    }
  }, [toast]);

  const getSourceStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle2 className="w-4 h-4 text-green-500" />;
      case 'failed':
        return <XCircle className="w-4 h-4 text-destructive" />;
      case 'crawling':
        return <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />;
      default:
        return <Clock className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const getSourceStatusBadge = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'success':
        return 'default';
      case 'failed':
        return 'destructive';
      case 'crawling':
        return 'secondary';
      default:
        return 'outline';
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) {
      return 'Never';
    }
    return new Date(dateString).toLocaleString();
  };

  return (
    <section className="flex-1 overflow-auto p-0 md:p-4">
      <div className="space-y-6">
        <Card className="border-white/[0.1]">
          <CardHeader className="gap-4 md:flex-row md:items-start md:justify-between">
            <div className="space-y-2">
              <Link to="/dashboard/insights" className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                <ArrowLeft className="w-3 h-3" />
                Back to Insights
              </Link>
              <div>
                <CardTitle className="text-xl">Insights Sources</CardTitle>
                <CardDescription>
                  Manage the sites that feed your insights pipeline, or search for new ones with AI when you need them.
                </CardDescription>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {isSearchMode && (
                <Button variant="outline" onClick={clearSearch}>
                  Show Sources Table
                </Button>
              )}
              <Button onClick={handleOpenCreate}>
                <Plus className="w-4 h-4 mr-2" />
                Add Source
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-2xl border border-white/[0.08] bg-black/30 p-4">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
                <div className="relative flex-1">
                  <Sparkles className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={queryInput}
                    onChange={(event) => setQueryInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') {
                        event.preventDefault();
                        handleSearch();
                      }
                    }}
                    placeholder="Search for new insight sources by topic, niche, or request..."
                    className="h-11 pl-9"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <Button onClick={handleSearch} disabled={!queryInput.trim() || recommendMutation.isPending}>
                    {recommendMutation.isPending ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Searching
                      </>
                    ) : (
                      <>
                        <ArrowUp className="w-4 h-4 mr-2" />
                        Search AI
                      </>
                    )}
                  </Button>
                  {isSearchMode && (
                    <Button variant="ghost" onClick={clearSearch}>
                      Clear
                    </Button>
                  )}
                </div>
              </div>
              {!isSearchMode && (
                <p className="mt-3 text-xs text-muted-foreground">
                  The table is the default view. Search only when you want AI recommendations for new sources.
                </p>
              )}
            </div>
          </CardContent>
        </Card>

        {isSearchMode ? (
          <SearchResultsSection
            activeSearchQuery={activeSearchQuery}
            results={recommendMutation.data}
            isSearching={recommendMutation.isPending}
            errorMessage={recommendMutation.error instanceof Error ? recommendMutation.error.message : undefined}
            selected={selectedRecommendations}
            statuses={recommendationStatuses}
            selectedCount={selectedRecommendationCount}
            isAddingSelection={isAddingRecommendations}
            onToggleSelected={handleToggleRecommendation}
            onToggleAll={handleToggleAllRecommendations}
            onAddOne={handleAddRecommendation}
            onAddSelected={handleAddSelectedRecommendations}
            onAddManually={handleOpenCreate}
          />
        ) : (
          <SourcesTableSection
            dataSources={dataSources}
            isLoading={isLoading}
            crawlMutationPending={crawlMutation.isPending}
            deletePending={deleteMutation.isPending}
            onAddSource={handleOpenCreate}
            onOpenEdit={handleOpenEdit}
            onTriggerCrawl={(id) => crawlMutation.mutate(id)}
            onDelete={(id) => deleteMutation.mutate(id)}
            formatDate={formatDate}
            getStatusBadge={getSourceStatusBadge}
            getStatusIcon={getSourceStatusIcon}
            onSearchRequested={() => {
              if (queryInput.trim()) {
                handleSearch();
              }
            }}
            hasSearchText={queryInput.trim().length > 0}
          />
        )}

        <WorkerControlsSection
          open={workerControlsOpen}
          onOpenChange={setWorkerControlsOpen}
          workerStatuses={workerStatuses}
          runningWorker={runningWorker}
          onRunWorker={handleRunWorker}
          onStopWorker={handleStopWorker}
        />
      </div>

      <Dialog open={isDialogOpen} onOpenChange={(open) => !open && handleCloseDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingSource ? 'Edit Insight Source' : 'Add Insight Source'}</DialogTitle>
            <DialogDescription>
              {editingSource
                ? 'Update the crawl settings for this source.'
                : 'Add a source manually when you already know exactly what to monitor.'}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="source-name">Name</Label>
              <Input
                id="source-name"
                placeholder="e.g. TechCrunch"
                value={formData.name}
                onChange={(event) => setFormData((prev) => ({ ...prev, name: event.target.value }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="source-url">URL</Label>
              <Input
                id="source-url"
                type="url"
                placeholder="https://example.com/blog"
                value={formData.url}
                onChange={(event) => setFormData((prev) => ({ ...prev, url: event.target.value }))}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="source-type">Type</Label>
                <Select
                  value={formData.source_type}
                  onValueChange={(value) => setFormData((prev) => ({ ...prev, source_type: value }))}
                >
                  <SelectTrigger id="source-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="blog">Blog</SelectItem>
                    <SelectItem value="news">News</SelectItem>
                    <SelectItem value="forum">Forum</SelectItem>
                    <SelectItem value="rss">RSS Feed</SelectItem>
                    <SelectItem value="newsletter">Newsletter</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="crawl-frequency">Crawl Frequency</Label>
                <Select
                  value={formData.crawl_frequency}
                  onValueChange={(value) => setFormData((prev) => ({ ...prev, crawl_frequency: value }))}
                >
                  <SelectTrigger id="crawl-frequency">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="hourly">Hourly</SelectItem>
                    <SelectItem value="daily">Daily</SelectItem>
                    <SelectItem value="weekly">Weekly</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="flex items-center justify-between rounded-xl border border-white/[0.08] p-3">
              <div className="space-y-0.5">
                <Label htmlFor="source-enabled">Enabled</Label>
                <p className="text-xs text-muted-foreground">Disable to pause crawling without removing the source.</p>
              </div>
              <Switch
                id="source-enabled"
                checked={formData.is_enabled}
                onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, is_enabled: checked }))}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={handleCloseDialog}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!formData.name.trim() || !formData.url.trim() || createMutation.isPending || updateMutation.isPending}
            >
              {createMutation.isPending || updateMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving
                </>
              ) : editingSource ? (
                'Update Source'
              ) : (
                'Add Source'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}

function SearchResultsSection({
  activeSearchQuery,
  results,
  isSearching,
  errorMessage,
  selected,
  statuses,
  selectedCount,
  isAddingSelection,
  onToggleSelected,
  onToggleAll,
  onAddOne,
  onAddSelected,
  onAddManually,
}: {
  activeSearchQuery: string;
  results?: RecommendDataSourcesResponse;
  isSearching: boolean;
  errorMessage?: string;
  selected: Record<string, boolean>;
  statuses: Record<string, RecommendationStatus>;
  selectedCount: number;
  isAddingSelection: boolean;
  onToggleSelected: (key: string, checked: boolean) => void;
  onToggleAll: (checked: boolean) => void;
  onAddOne: (recommendation: DataSourceRecommendation) => void;
  onAddSelected: () => void;
  onAddManually: () => void;
}) {
  const recommendations = results?.recommendations || [];
  const selectableRecommendations = recommendations.filter((recommendation) => statuses[recommendationKey(recommendation)]?.state !== 'added');
  const allSelected = selectableRecommendations.length > 0 && selectableRecommendations.every((recommendation) => selected[recommendationKey(recommendation)]);

  return (
    <Card>
      <CardHeader className="gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <CardTitle className="text-base">AI Search Results</CardTitle>
          <CardDescription>
            {`Recommendations for "${activeSearchQuery}"`}
          </CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={onAddManually}>
            <Plus className="w-4 h-4 mr-2" />
            Add Manually
          </Button>
          <Button variant="outline" onClick={onAddSelected} disabled={selectedCount === 0 || isAddingSelection}>
            {isAddingSelection ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Adding
              </>
            ) : (
              <>
                <Plus className="w-4 h-4 mr-2" />
                Add Selected ({selectedCount})
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {isSearching ? (
          <div className="flex items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="w-5 h-5 animate-spin" />
            <span className="ml-2">Searching for source recommendations...</span>
          </div>
        ) : errorMessage ? (
          <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive flex items-start gap-2">
            <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <span>{errorMessage}</span>
          </div>
        ) : recommendations.length === 0 ? (
          <div className="rounded-xl border border-dashed py-16 text-center text-muted-foreground">
            <Sparkles className="mx-auto mb-3 h-8 w-8 opacity-60" />
            <p className="font-medium text-foreground">No matching sources found</p>
            <p className="mt-1 text-sm">Try a narrower search or add a source manually.</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    checked={allSelected}
                    onCheckedChange={(checked) => onToggleAll(checked === true)}
                    aria-label="Select all recommendations"
                  />
                </TableHead>
                <TableHead>Source</TableHead>
                <TableHead>Why it matches</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recommendations.map((recommendation) => {
                const key = recommendationKey(recommendation);
                const status = statuses[key];
                const isAdded = status?.state === 'added';

                return (
                  <TableRow key={key} data-state={selected[key] ? 'selected' : undefined}>
                    <TableCell>
                      <Checkbox
                        checked={selected[key] || false}
                        disabled={isAdded || status?.state === 'adding'}
                        onCheckedChange={(checked) => onToggleSelected(key, checked === true)}
                        aria-label={`Select ${recommendation.name}`}
                      />
                    </TableCell>
                    <TableCell className="whitespace-normal">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-medium">{recommendation.name}</span>
                          <Badge variant="outline" className="text-[11px]">
                            {recommendation.domain}
                          </Badge>
                          {isAdded && (
                            <Badge variant="secondary" className="text-[11px]">
                              <CheckCircle className="w-3 h-3 mr-1" />
                              Added
                            </Badge>
                          )}
                        </div>
                        <a
                          href={recommendation.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                        >
                          <Globe className="w-3 h-3" />
                          <span className="truncate">{recommendation.url}</span>
                        </a>
                        {recommendation.sample_url && recommendation.sample_url !== recommendation.url && (
                          <a
                            href={recommendation.sample_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                          >
                            <ExternalLink className="w-3 h-3" />
                            <span className="truncate">
                              Sample result: {recommendation.sample_title || recommendation.sample_url}
                            </span>
                          </a>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="whitespace-normal">
                      <div className="space-y-1">
                        {recommendation.reason && (
                          <p className="text-sm">{recommendation.reason}</p>
                        )}
                        {recommendation.summary && (
                          <p className="text-xs text-muted-foreground line-clamp-3">{recommendation.summary}</p>
                        )}
                        {status?.state === 'error' && (
                          <p className="text-xs text-destructive">{status.message || 'Failed to add source'}</p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getRecommendationTypeVariant(recommendation.source_type)} className="capitalize">
                        {recommendation.source_type}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant={isAdded ? 'outline' : 'default'}
                        size="sm"
                        disabled={isAdded || status?.state === 'adding'}
                        onClick={() => onAddOne(recommendation)}
                      >
                        {status?.state === 'adding' ? (
                          <>
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            Adding
                          </>
                        ) : isAdded ? (
                          'Added'
                        ) : (
                          'Add Source'
                        )}
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function SourcesTableSection({
  dataSources,
  isLoading,
  crawlMutationPending,
  deletePending,
  onAddSource,
  onOpenEdit,
  onTriggerCrawl,
  onDelete,
  formatDate,
  getStatusBadge,
  getStatusIcon,
  onSearchRequested,
  hasSearchText,
}: {
  dataSources: DataSource[];
  isLoading: boolean;
  crawlMutationPending: boolean;
  deletePending: boolean;
  onAddSource: () => void;
  onOpenEdit: (source: DataSource) => void;
  onTriggerCrawl: (id: string) => void;
  onDelete: (id: string) => void;
  formatDate: (dateString?: string) => string;
  getStatusBadge: (status: string) => 'default' | 'secondary' | 'destructive' | 'outline';
  getStatusIcon: (status: string) => React.ReactNode;
  onSearchRequested: () => void;
  hasSearchText: boolean;
}) {
  return (
    <Card>
      <CardHeader className="gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <CardTitle className="text-base">Current Sources</CardTitle>
          <CardDescription>
            Your insight pipeline sources, crawl status, and source management actions.
          </CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={onSearchRequested} disabled={!hasSearchText}>
            <Sparkles className="w-4 h-4 mr-2" />
            Search Sources
          </Button>
          <Button onClick={onAddSource}>
            <Plus className="w-4 h-4 mr-2" />
            Add Source
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="w-5 h-5 animate-spin" />
            <span className="ml-2">Loading sources...</span>
          </div>
        ) : dataSources.length === 0 ? (
          <div className="rounded-xl border border-dashed py-16 text-center text-muted-foreground">
            <Globe className="mx-auto mb-3 h-10 w-10 opacity-60" />
            <p className="font-medium text-foreground">No insight sources yet</p>
            <p className="mt-1 text-sm">Add a source manually or search above for new recommendations.</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Cadence</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Items</TableHead>
                <TableHead>Last Crawled</TableHead>
                <TableHead>Next Crawl</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dataSources.map((source) => (
                <TableRow key={source.id}>
                  <TableCell className="whitespace-normal">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium">{source.name}</span>
                        {source.is_discovered && (
                          <Badge variant="secondary" className="text-[11px]">Discovered</Badge>
                        )}
                      </div>
                      <a
                        href={source.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                      >
                        <ExternalLink className="w-3 h-3" />
                        <span className="truncate">{source.url}</span>
                      </a>
                      {source.error_message && (
                        <p className="text-xs text-destructive line-clamp-2">{source.error_message}</p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="capitalize">{source.source_type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="capitalize">{source.crawl_frequency}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadge(source.crawl_status)} className="flex w-fit items-center gap-1 capitalize">
                      {getStatusIcon(source.crawl_status)}
                      {source.crawl_status}
                    </Badge>
                  </TableCell>
                  <TableCell>{source.content_count}</TableCell>
                  <TableCell>{formatDate(source.last_crawled_at)}</TableCell>
                  <TableCell>{formatDate(source.next_crawl_at)}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => onTriggerCrawl(source.id)}
                        disabled={crawlMutationPending || source.crawl_status === 'crawling'}
                      >
                        <RefreshCw className={`w-4 h-4 ${source.crawl_status === 'crawling' ? 'animate-spin' : ''}`} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => onOpenEdit(source)}
                      >
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8" disabled={deletePending}>
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Delete source</AlertDialogTitle>
                            <AlertDialogDescription>
                              Are you sure you want to delete "{source.name}"? This also removes crawled content for that source.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => onDelete(source.id)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              Delete
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function WorkerControlsSection({
  open,
  onOpenChange,
  workerStatuses,
  runningWorker,
  onRunWorker,
  onStopWorker,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerStatuses: Record<string, WorkerStatus>;
  runningWorker: string | null;
  onRunWorker: (name: string) => void;
  onStopWorker: (name: string) => void;
}) {
  const workers = ['crawl', 'insight', 'discovery'];

  return (
    <Collapsible open={open} onOpenChange={onOpenChange}>
      <Card className="gap-0">
        <CollapsibleTrigger asChild>
          <button className="flex w-full items-center justify-between px-6 py-5 text-left">
            <div>
              <div className="flex items-center gap-2">
                <Zap className="w-4 h-4 text-primary" />
                <span className="font-medium">Pipeline Controls</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                Secondary worker controls for crawling, insight generation, and discovery.
              </p>
            </div>
            <ChevronDown className={`w-4 h-4 transition-transform ${open ? 'rotate-180' : ''}`} />
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className="pb-6">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              {workers.map((workerName) => {
                const status = workerStatuses[workerName];
                const IconComponent = workerIcons[workerName] || RefreshCw;
                const isRunning = status?.state === 'running';
                const isMutating = runningWorker === workerName;

                return (
                  <div
                    key={workerName}
                    className={`rounded-xl border p-4 ${getWorkerStateBgColor(status?.state || 'idle')}`}
                  >
                    <div className="mb-2 flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <IconComponent className={`w-4 h-4 ${isRunning ? 'animate-spin' : ''} ${getWorkerStateColor(status?.state || 'idle')}`} />
                        <span className="font-medium text-sm">{getWorkerDisplayName(workerName)}</span>
                      </div>
                      {isRunning ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onStopWorker(workerName)}
                          disabled={isMutating}
                        >
                          {isMutating ? (
                            <Loader2 className="w-3 h-3 animate-spin" />
                          ) : (
                            <>
                              <Square className="w-3 h-3 mr-1" />
                              Stop
                            </>
                          )}
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onRunWorker(workerName)}
                          disabled={isMutating}
                        >
                          {isMutating ? (
                            <Loader2 className="w-3 h-3 animate-spin" />
                          ) : (
                            <>
                              <Play className="w-3 h-3 mr-1" />
                              Run
                            </>
                          )}
                        </Button>
                      )}
                    </div>
                    <p className="mb-2 text-xs text-muted-foreground">{getWorkerDescription(workerName)}</p>
                    {status && (
                      <div className="space-y-2">
                        {isRunning && (
                          <>
                            <Progress value={status.progress} className="h-1.5" />
                            <p className="text-xs text-muted-foreground">
                              {status.message || 'Processing...'}
                              {status.items_total > 0 && (
                                <span className="ml-1">({status.items_done}/{status.items_total})</span>
                              )}
                            </p>
                          </>
                        )}
                        {status.state === 'completed' && status.completed_at && (
                          <p className="text-xs text-muted-foreground">
                            Completed: {new Date(status.completed_at).toLocaleString()}
                          </p>
                        )}
                        {status.state === 'failed' && status.error && (
                          <p className="flex items-center gap-1 text-xs text-destructive">
                            <AlertCircle className="w-3 h-3" />
                            {status.error}
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
