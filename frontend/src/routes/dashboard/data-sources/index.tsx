import { useState, useEffect, useCallback, useRef } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { 
  Globe, 
  Plus, 
  Pencil, 
  Trash2, 
  Loader2, 
  RefreshCw, 
  Clock, 
  CheckCircle2, 
  XCircle, 
  AlertCircle,
  ExternalLink,
  Play,
  Square,
  Sparkles,
  Search,
  Zap,
  ArrowUp,
  CheckCircle,
  CircleAlert,
  Link2,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Progress } from '@/components/ui/progress';
import { Checkbox } from '@/components/ui/checkbox';
import {
  PromptInput,
  PromptInputAction,
  PromptInputActions,
  PromptInputTextarea,
} from '@/components/ui/prompt-input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useToast } from '@/hooks/use-toast';
import { ApiError } from '@/services/authenticatedFetch';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { 
  listDataSources, 
  createDataSource, 
  updateDataSource, 
  deleteDataSource,
  recommendDataSources,
  triggerCrawl,
  type DataSource,
  type DataSourceRecommendation,
  type CreateDataSourceRequest,
  type RecommendDataSourcesResponse,
  type UpdateDataSourceRequest,
} from '@/services/dataSources';
import {
  getWorkersStatus,
  runWorker,
  stopWorker,
  getWorkerDisplayName,
  getWorkerDescription,
  type WorkerStatus,
  type WorkerState,
} from '@/services/workers';
import { VITE_WS_URL } from '@/services/constants';

export const Route = createFileRoute('/dashboard/data-sources/')({
  component: DataSourcesPage,
});

// Worker icon mapping
const WorkerIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  crawl: RefreshCw,
  insight: Sparkles,
  discovery: Search,
};

// Worker status colors
const getStateColor = (state: WorkerState): string => {
  switch (state) {
    case 'running': return 'text-blue-500';
    case 'completed': return 'text-green-500';
    case 'failed': return 'text-destructive';
    default: return 'text-muted-foreground';
  }
};

const getStateBgColor = (state: WorkerState): string => {
  switch (state) {
    case 'running': return 'bg-blue-500/10';
    case 'completed': return 'bg-green-500/10';
    case 'failed': return 'bg-destructive/10';
    default: return 'bg-muted';
  }
};

type RecommendationSaveState = 'idle' | 'adding' | 'added' | 'error';

interface RecommendationStatus {
  state: RecommendationSaveState;
  message?: string;
}

const getRecommendationStatusLabel = (status?: RecommendationStatus) => {
  switch (status?.state) {
    case 'adding':
      return 'Adding...';
    case 'added':
      return 'Added';
    case 'error':
      return status.message || 'Failed';
    default:
      return '';
  }
};

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

// Worker Controls Component
interface WorkerControlsProps {
  workerStatuses: Record<string, WorkerStatus>;
  onRunWorker: (name: string) => void;
  onStopWorker: (name: string) => void;
  runningMutation: string | null;
}

interface SourceRecommendationsPanelProps {
  query: string;
  onQueryChange: (value: string) => void;
  onSearch: () => void;
  isSearching: boolean;
  results?: RecommendDataSourcesResponse;
  searchError?: string;
  selected: Record<string, boolean>;
  statuses: Record<string, RecommendationStatus>;
  onToggleSelected: (key: string, checked: boolean) => void;
  onToggleAll: (checked: boolean) => void;
  onAddOne: (recommendation: DataSourceRecommendation) => void;
  onAddSelected: () => void;
  selectedCount: number;
  isAddingSelection: boolean;
  onManualAdd: () => void;
}

function SourceRecommendationsPanel({
  query,
  onQueryChange,
  onSearch,
  isSearching,
  results,
  searchError,
  selected,
  statuses,
  onToggleSelected,
  onToggleAll,
  onAddOne,
  onAddSelected,
  selectedCount,
  isAddingSelection,
  onManualAdd,
}: SourceRecommendationsPanelProps) {
  const recommendations = results?.recommendations || [];
  const selectableRecommendations = recommendations.filter((recommendation) => statuses[recommendationKey(recommendation)]?.state !== 'added');
  const allSelected = selectableRecommendations.length > 0 && selectableRecommendations.every((recommendation) => selected[recommendationKey(recommendation)]);

  return (
    <Card className="mb-6 border-primary/20">
      <CardHeader className="pb-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-primary" />
              <CardTitle className="text-base">AI Source Finder</CardTitle>
            </div>
            <CardDescription>
              Describe a topic or request and get ranked source recommendations you can add to your crawl list.
            </CardDescription>
          </div>
          <Button variant="outline" onClick={onManualAdd}>
            <Plus className="w-4 h-4 mr-2" />
            Add Manually
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <PromptInput
          value={query}
          onValueChange={onQueryChange}
          onSubmit={onSearch}
          isLoading={isSearching}
          className="rounded-2xl border-primary/20"
        >
          <PromptInputTextarea
            placeholder="Find sources for topics like AI coding agents, crypto market structure, or developer tooling news..."
            className="min-h-[72px]"
          />
          <PromptInputActions className="justify-end pt-2">
            <PromptInputAction tooltip={isSearching ? 'Searching' : 'Search for sources'}>
              <Button
                variant="default"
                size="icon"
                className="h-8 w-8 rounded-full"
                disabled={!query.trim() || isSearching}
                onClick={onSearch}
              >
                {isSearching ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <ArrowUp className="size-4" />
                )}
              </Button>
            </PromptInputAction>
          </PromptInputActions>
        </PromptInput>

        {searchError && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive flex items-start gap-2">
            <CircleAlert className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <span>{searchError}</span>
          </div>
        )}

        {results && (
          <div className="space-y-3">
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <div>
                <p className="text-sm font-medium">
                  {recommendations.length === 0
                    ? 'No recommendations found'
                    : `Recommended sources for "${results.query}"`}
                </p>
                <p className="text-xs text-muted-foreground">
                  Review the list before creating data sources. Recommendations stay temporary until you add them.
                </p>
              </div>
              {recommendations.length > 0 && (
                <Button
                  variant="outline"
                  onClick={onAddSelected}
                  disabled={selectedCount === 0 || isAddingSelection}
                >
                  {isAddingSelection ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Adding Selected
                    </>
                  ) : (
                    <>
                      <Plus className="w-4 h-4 mr-2" />
                      Add Selected ({selectedCount})
                    </>
                  )}
                </Button>
              )}
            </div>

            {recommendations.length === 0 ? (
              <div className="rounded-lg border border-dashed py-8 text-center text-sm text-muted-foreground">
                Try a more specific request, a narrower niche, or a different keyword set.
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
                              {status?.state === 'added' && (
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
                              className="inline-flex max-w-full items-center gap-1 text-xs text-primary hover:underline"
                            >
                              <Link2 className="w-3 h-3" />
                              <span className="truncate">{recommendation.url}</span>
                            </a>
                            {recommendation.sample_url && recommendation.sample_url !== recommendation.url && (
                              <a
                                href={recommendation.sample_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="inline-flex max-w-full items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
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
                              <p className="text-xs text-destructive">{getRecommendationStatusLabel(status)}</p>
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
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function WorkerControls({ workerStatuses, onRunWorker, onStopWorker, runningMutation }: WorkerControlsProps) {
  const workers = ['crawl', 'insight', 'discovery'];

  return (
    <Card className="mb-6">
      <CardHeader className="pb-3">
        <div className="flex items-center gap-2">
          <Zap className="w-5 h-5 text-primary" />
          <CardTitle className="text-base">Worker Controls</CardTitle>
        </div>
        <CardDescription>
          Run data processing workers manually or view their status.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {workers.map((workerName) => {
            const status = workerStatuses[workerName];
            const IconComponent = WorkerIcons[workerName] || RefreshCw;
            const isRunning = status?.state === 'running';
            const isMutating = runningMutation === workerName;

            return (
              <div
                key={workerName}
                className={`p-4 rounded-lg border ${getStateBgColor(status?.state || 'idle')}`}
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <IconComponent className={`w-4 h-4 ${isRunning ? 'animate-spin' : ''} ${getStateColor(status?.state || 'idle')}`} />
                    <span className="font-medium text-sm">{getWorkerDisplayName(workerName)}</span>
                  </div>
                  {isRunning ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onStopWorker(workerName)}
                      disabled={isMutating}
                      className="h-7 px-2"
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
                      className="h-7 px-2"
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

                <p className="text-xs text-muted-foreground mb-2">
                  {getWorkerDescription(workerName)}
                </p>

                {status && (
                  <div className="space-y-2">
                    {isRunning && (
                      <>
                        <Progress value={status.progress} className="h-1.5" />
                        <p className="text-xs text-muted-foreground">
                          {status.message || 'Processing...'}
                          {status.items_total > 0 && (
                            <span className="ml-1">
                              ({status.items_done}/{status.items_total})
                            </span>
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
                      <p className="text-xs text-destructive flex items-center gap-1">
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
    </Card>
  );
}

function DataSourcesPage() {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<DataSource | null>(null);
  const [formData, setFormData] = useState<CreateDataSourceRequest>({
    name: '',
    url: '',
    source_type: 'blog',
    crawl_frequency: 'daily',
    is_enabled: true,
  });
  const [recommendationQuery, setRecommendationQuery] = useState('');
  const [selectedRecommendations, setSelectedRecommendations] = useState<Record<string, boolean>>({});
  const [recommendationStatuses, setRecommendationStatuses] = useState<Record<string, RecommendationStatus>>({});
  const [isAddingRecommendations, setIsAddingRecommendations] = useState(false);
  const [workerStatuses, setWorkerStatuses] = useState<Record<string, WorkerStatus>>({});
  const [runningMutation, setRunningMutation] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();

  useEffect(() => {
    setPageTitle("Data Sources");
  }, [setPageTitle]);

  // Fetch initial worker statuses
  useEffect(() => {
    getWorkersStatus().then((response) => {
      const statuses: Record<string, WorkerStatus> = {};
      response.workers.forEach((w) => {
        statuses[w.name] = w;
      });
      setWorkerStatuses(statuses);
    }).catch(console.error);
  }, []);

  // WebSocket connection for live updates
  useEffect(() => {
    const wsUrl = VITE_WS_URL || `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/websocket`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      // Subscribe to worker status channel
      ws.send(JSON.stringify({
        action: 'subscribe',
        channel: 'worker-status',
      }));
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        if (data.type === 'worker-status') {
          setWorkerStatuses((prev) => ({
            ...prev,
            [data.worker_name]: data.status,
          }));

          // Show toast for state changes
          if (data.status.state === 'completed') {
            toast({
              title: `${getWorkerDisplayName(data.worker_name)} Completed`,
              description: data.status.message || 'Worker completed successfully',
            });
            // Refresh data sources after crawl completes
            if (data.worker_name === 'crawl') {
              queryClient.invalidateQueries({ queryKey: ['data-sources'] });
            }
          } else if (data.status.state === 'failed') {
            toast({
              title: `${getWorkerDisplayName(data.worker_name)} Failed`,
              description: data.status.error || 'Worker failed',
              variant: 'destructive',
            });
          }
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
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

  // Worker control handlers
  const handleRunWorker = useCallback(async (name: string) => {
    setRunningMutation(name);
    try {
      await runWorker(name);
      toast({
        title: 'Worker Started',
        description: `${getWorkerDisplayName(name)} is now running`,
      });
    } catch (error) {
      toast({
        title: 'Failed to Start Worker',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      });
    } finally {
      setRunningMutation(null);
    }
  }, [toast]);

  const handleStopWorker = useCallback(async (name: string) => {
    setRunningMutation(name);
    try {
      await stopWorker(name);
      toast({
        title: 'Worker Stopped',
        description: `${getWorkerDisplayName(name)} has been stopped`,
      });
    } catch (error) {
      toast({
        title: 'Failed to Stop Worker',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      });
    } finally {
      setRunningMutation(null);
    }
  }, [toast]);

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
          description: 'Try a more specific topic or add a source manually.',
        });
      }
    },
    onError: (error) => {
      toast({
        title: 'Source search failed',
        description: error instanceof Error ? error.message : 'Unable to search for source recommendations.',
        variant: 'destructive',
      });
    },
  });

  // Handle both array and paginated response
  const dataSources: DataSource[] = Array.isArray(data) 
    ? data 
    : (data as any)?.data_sources || [];
  const recommendations = recommendMutation.data?.recommendations || [];
  const selectedRecommendationCount = recommendations.filter((recommendation) => {
    const key = recommendationKey(recommendation);
    return selectedRecommendations[key] && recommendationStatuses[key]?.state !== 'added';
  }).length;

  const createMutation = useMutation({
    mutationFn: createDataSource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Success', description: 'Data source created successfully' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to create data source', variant: 'destructive' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateDataSourceRequest }) => updateDataSource(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Success', description: 'Data source updated successfully' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to update data source', variant: 'destructive' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteDataSource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Success', description: 'Data source deleted successfully' });
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to delete data source', variant: 'destructive' });
    },
  });

  const crawlMutation = useMutation({
    mutationFn: triggerCrawl,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-sources'] });
      toast({ title: 'Success', description: 'Crawl triggered successfully' });
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to trigger crawl', variant: 'destructive' });
    },
  });

  const handleOpenCreate = () => {
    setEditingSource(null);
    setFormData({
      name: '',
      url: '',
      source_type: 'blog',
      crawl_frequency: 'daily',
      is_enabled: true,
    });
    setIsDialogOpen(true);
  };

  const handleOpenEdit = (source: DataSource) => {
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
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setEditingSource(null);
  };

  const handleSearchRecommendations = useCallback(() => {
    const trimmedQuery = recommendationQuery.trim();
    if (!trimmedQuery || recommendMutation.isPending) {
      return;
    }

    recommendMutation.mutate({
      query: trimmedQuery,
      limit: 8,
    });
  }, [recommendationQuery, recommendMutation]);

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
        const reason = result.reason instanceof ApiError
          ? result.reason.message
          : result.reason instanceof Error
            ? result.reason.message
            : 'Failed to add source';
        next[key] = { state: 'error', message: reason };
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
          : 'Recommended sources are now in your crawl list.',
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

  const handleSubmit = () => {
    if (editingSource) {
      updateMutation.mutate({ id: editingSource.id, data: formData });
    } else {
      createMutation.mutate(formData);
    }
  };

  const getStatusIcon = (status: string) => {
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

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
      success: 'default',
      failed: 'destructive',
      crawling: 'secondary',
      pending: 'outline',
    };
    return variants[status] || 'outline';
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleString();
  };

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto">
      <SourceRecommendationsPanel
        query={recommendationQuery}
        onQueryChange={setRecommendationQuery}
        onSearch={handleSearchRecommendations}
        isSearching={recommendMutation.isPending}
        results={recommendMutation.data}
        searchError={recommendMutation.error instanceof Error ? recommendMutation.error.message : undefined}
        selected={selectedRecommendations}
        statuses={recommendationStatuses}
        onToggleSelected={handleToggleRecommendation}
        onToggleAll={handleToggleAllRecommendations}
        onAddOne={handleAddRecommendation}
        onAddSelected={handleAddSelectedRecommendations}
        selectedCount={selectedRecommendationCount}
        isAddingSelection={isAddingRecommendations}
        onManualAdd={handleOpenCreate}
      />

      {/* Worker Controls */}
      <WorkerControls
        workerStatuses={workerStatuses}
        onRunWorker={handleRunWorker}
        onStopWorker={handleStopWorker}
        runningMutation={runningMutation}
      />

      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-lg font-semibold">Data Sources</h2>
          <p className="text-sm text-muted-foreground">
            Configure websites and blogs to crawl for insights.
          </p>
        </div>
        <Button onClick={handleOpenCreate}>
          <Plus className="w-4 h-4 mr-2" />
          Add Source
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 animate-spin" />
          <span className="ml-2">Loading data sources...</span>
        </div>
      ) : dataSources.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <Globe className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg font-medium mb-2">No data sources yet</p>
          <p className="text-sm mb-4">Use the AI source finder above to build a crawl list, or add one manually.</p>
          <div className="flex items-center justify-center gap-2">
            <Button onClick={handleSearchRecommendations} disabled={!recommendationQuery.trim() || recommendMutation.isPending}>
              <Sparkles className="w-4 h-4 mr-2" />
              Find Sources
            </Button>
            <Button variant="outline" onClick={handleOpenCreate}>
              <Plus className="w-4 h-4 mr-2" />
              Add Manually
            </Button>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {dataSources.map((source) => (
            <Card key={source.id} className={!source.is_enabled ? 'opacity-60' : ''}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <Globe className="w-4 h-4 text-primary" />
                    <CardTitle className="text-base line-clamp-1">{source.name}</CardTitle>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => crawlMutation.mutate(source.id)}
                      disabled={crawlMutation.isPending || source.crawl_status === 'crawling'}
                    >
                      <RefreshCw className={`w-3 h-3 ${source.crawl_status === 'crawling' ? 'animate-spin' : ''}`} />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => handleOpenEdit(source)}
                    >
                      <Pencil className="w-3 h-3" />
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-7 w-7">
                          <Trash2 className="w-3 h-3" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete Data Source</AlertDialogTitle>
                          <AlertDialogDescription>
                            Are you sure you want to delete "{source.name}"? This will also delete all crawled content.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => deleteMutation.mutate(source.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
                <a
                  href={source.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-primary hover:underline flex items-center gap-1"
                >
                  <ExternalLink className="w-3 h-3" />
                  <span className="truncate">{source.url}</span>
                </a>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2 mb-3 flex-wrap">
                  <Badge variant="outline" className="text-xs capitalize">
                    {source.source_type}
                  </Badge>
                  <Badge variant="outline" className="text-xs capitalize">
                    {source.crawl_frequency}
                  </Badge>
                  <Badge variant={getStatusBadge(source.crawl_status)} className="text-xs flex items-center gap-1">
                    {getStatusIcon(source.crawl_status)}
                    {source.crawl_status}
                  </Badge>
                </div>

                <div className="space-y-1 text-xs text-muted-foreground">
                  <div className="flex items-center justify-between">
                    <span>Content items:</span>
                    <span className="font-medium">{source.content_count}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Last crawled:</span>
                    <span>{formatDate(source.last_crawled_at)}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Next crawl:</span>
                    <span>{formatDate(source.next_crawl_at)}</span>
                  </div>
                </div>

                {source.error_message && (
                  <div className="mt-2 p-2 bg-destructive/10 rounded text-xs text-destructive flex items-start gap-1">
                    <AlertCircle className="w-3 h-3 mt-0.5 flex-shrink-0" />
                    <span className="line-clamp-2">{source.error_message}</span>
                  </div>
                )}

                {source.is_discovered && (
                  <Badge variant="secondary" className="mt-2 text-xs">
                    Auto-discovered
                  </Badge>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={(open) => !open && handleCloseDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingSource ? 'Edit Data Source' : 'Add Data Source'}</DialogTitle>
            <DialogDescription>
              {editingSource 
                ? 'Update the data source configuration.'
                : 'Add a website or blog to crawl for insights.'}
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="e.g., TechCrunch, Hacker News"
                value={formData.name}
                onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
              />
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="url">URL</Label>
              <Input
                id="url"
                type="url"
                placeholder="https://example.com/blog"
                value={formData.url}
                onChange={(e) => setFormData((prev) => ({ ...prev, url: e.target.value }))}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="source_type">Type</Label>
                <Select
                  value={formData.source_type}
                  onValueChange={(value) => setFormData((prev) => ({ ...prev, source_type: value }))}
                >
                  <SelectTrigger>
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
                <Label htmlFor="crawl_frequency">Crawl Frequency</Label>
                <Select
                  value={formData.crawl_frequency}
                  onValueChange={(value) => setFormData((prev) => ({ ...prev, crawl_frequency: value }))}
                >
                  <SelectTrigger>
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

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="is_enabled">Enabled</Label>
                <p className="text-xs text-muted-foreground">
                  Disable to pause crawling without deleting the source.
                </p>
              </div>
              <Switch
                id="is_enabled"
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
              {(createMutation.isPending || updateMutation.isPending) ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving...
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
