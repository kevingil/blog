import { useEffect, useMemo, useRef, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight,
  Lightbulb,
  Tag,
  Calendar,
  Pin,
  Check,
  Clock3,
  Loader2,
  Play,
  RefreshCw,
  Search,
  Sparkles,
  Square,
  WandSparkles,
  XCircle,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { useToast } from "@/hooks/use-toast";
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import {
  listInsights,
  listTopics,
  markInsightAsRead,
  toggleInsightPinned,
  searchInsights,
  type Insight,
} from "@/services/insights";
import { useWorkerStatuses } from "@/hooks/use-worker-statuses";
import {
  PIPELINE_WORKER_NAME,
  getWorkerDescription,
  getWorkerDisplayName,
  runWorker,
  stopWorker,
  type WorkerState,
  type WorkerStatus,
} from "@/services/workers";

export const Route = createFileRoute("/dashboard/insights/")({
  component: InsightsPage,
});

function InsightsPage() {
  const [page, setPage] = useState(1);
  const [selectedTopicId, setSelectedTopicId] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [isWorkflowDialogOpen, setIsWorkflowDialogOpen] = useState(false);
  const [workflowAction, setWorkflowAction] = useState<string | null>(null);
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();
  const workerStatuses = useWorkerStatuses();
  const previousStatusesRef = useRef<Record<string, WorkerStatus>>({});

  useEffect(() => {
    setPageTitle("Insights");
  }, [setPageTitle]);

  useEffect(() => {
    const watchedWorkers = [PIPELINE_WORKER_NAME, "crawl", "insight"];

    watchedWorkers.forEach((workerName) => {
      const previousStatus = previousStatusesRef.current[workerName];
      const nextStatus = workerStatuses[workerName];

      if (!nextStatus || previousStatus?.state === nextStatus.state) {
        return;
      }

      if (nextStatus.state === "completed") {
        toast({
          title: `${getWorkerDisplayName(workerName)} completed`,
          description:
            nextStatus.message || "Workflow step completed successfully.",
        });
        queryClient.invalidateQueries({ queryKey: ["insights"] });
      }

      if (nextStatus.state === "failed") {
        toast({
          title: `${getWorkerDisplayName(workerName)} failed`,
          description:
            nextStatus.error || nextStatus.message || "Workflow step failed.",
          variant: "destructive",
        });
      }
    });

    previousStatusesRef.current = workerStatuses;
  }, [queryClient, toast, workerStatuses]);

  // Load topics
  const { data: topics = [] } = useQuery({
    queryKey: ["insight-topics"],
    queryFn: listTopics,
  });

  // Load insights
  const { data: insightsData, isLoading } = useQuery({
    queryKey: ["insights", page, selectedTopicId],
    queryFn: () =>
      listInsights(
        page,
        12,
        selectedTopicId === "all" ? undefined : selectedTopicId,
      ),
  });

  // Search insights
  const { data: searchResults, isLoading: isSearchLoading } = useQuery({
    queryKey: ["insights-search", searchQuery],
    queryFn: () => searchInsights(searchQuery, 20),
    enabled: searchQuery.length > 2,
  });

  const insights =
    searchQuery.length > 2 ? searchResults || [] : insightsData?.insights || [];
  const total = insightsData?.total || 0;
  const totalPages = Math.ceil(total / 12);
  const workflowStatuses = useMemo(
    () => ({
      pipeline: workerStatuses[PIPELINE_WORKER_NAME],
      crawl: workerStatuses["crawl"],
      insight: workerStatuses["insight"],
    }),
    [workerStatuses],
  );

  // Mutations
  const markReadMutation = useMutation({
    mutationFn: markInsightAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["insights"] });
    },
  });

  const togglePinMutation = useMutation({
    mutationFn: toggleInsightPinned,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["insights"] });
      toast({ title: "Success", description: "Insight pin status updated" });
    },
  });

  const handleMarkAsRead = async (insightId: string) => {
    markReadMutation.mutate(insightId);
  };

  const handleTogglePin = async (e: React.MouseEvent, insightId: string) => {
    e.stopPropagation();
    togglePinMutation.mutate(insightId);
  };

  const handleWorkerAction = async (
    workerName: string,
    action: "run" | "stop",
  ) => {
    setWorkflowAction(`${action}:${workerName}`);
    try {
      if (action === "run") {
        await runWorker(workerName);
        toast({
          title: `${getWorkerDisplayName(workerName)} started`,
          description:
            workerName === PIPELINE_WORKER_NAME
              ? "The full insights pipeline is now running."
              : `${getWorkerDisplayName(workerName)} is now running.`,
        });
      } else {
        await stopWorker(workerName);
        toast({
          title: `${getWorkerDisplayName(workerName)} stopped`,
          description: `${getWorkerDisplayName(workerName)} has been stopped.`,
        });
      }
    } catch (error) {
      toast({
        title:
          action === "run"
            ? "Failed to start workflow"
            : "Failed to stop workflow",
        description: error instanceof Error ? error.message : "Unknown error.",
        variant: "destructive",
      });
    } finally {
      setWorkflowAction(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
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
        <Select
          value={selectedTopicId}
          onValueChange={(value) => {
            setSelectedTopicId(value);
            setPage(1);
          }}
        >
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
        <Link to="/dashboard/insights/sources">
          <Button variant="outline">
            <Search className="w-4 h-4 mr-2" />
            Sources
          </Button>
        </Link>
        <Button onClick={() => setIsWorkflowDialogOpen(true)}>
          <WandSparkles className="w-4 h-4 mr-2" />
          Generate Insights
        </Button>
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
          <p className="text-sm">
            Add sources and run the pipeline to start generating insights.
          </p>
          <div className="mt-4 flex items-center justify-center gap-2">
            <Button onClick={() => setIsWorkflowDialogOpen(true)}>
              <WandSparkles className="w-4 h-4 mr-2" />
              Generate Insights
            </Button>
            <Link to="/dashboard/insights/sources">
              <Button variant="outline">
                <Search className="w-4 h-4 mr-2" />
                Manage Sources
              </Button>
            </Link>
          </div>
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
                      className={
                        page === 1
                          ? "pointer-events-none opacity-50"
                          : "cursor-pointer"
                      }
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
                      className={
                        page === totalPages
                          ? "pointer-events-none opacity-50"
                          : "cursor-pointer"
                      }
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </>
      )}

      <InsightsWorkflowDialog
        open={isWorkflowDialogOpen}
        onOpenChange={setIsWorkflowDialogOpen}
        workerStatuses={workflowStatuses}
        activeAction={workflowAction}
        onRun={(workerName) => handleWorkerAction(workerName, "run")}
        onStop={(workerName) => handleWorkerAction(workerName, "stop")}
      />
    </section>
  );
}

function InsightsWorkflowDialog({
  open,
  onOpenChange,
  workerStatuses,
  activeAction,
  onRun,
  onStop,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerStatuses: {
    pipeline?: WorkerStatus;
    crawl?: WorkerStatus;
    insight?: WorkerStatus;
  };
  activeAction: string | null;
  onRun: (workerName: string) => void;
  onStop: (workerName: string) => void;
}) {
  const pipelineStatus = workerStatuses.pipeline;
  const crawlStatus = workerStatuses.crawl;
  const insightStatus = workerStatuses.insight;
  const pipelineRunning = pipelineStatus?.state === "running";
  const workflowBusy =
    pipelineRunning ||
    crawlStatus?.state === "running" ||
    insightStatus?.state === "running";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[min(94vw,72rem)] max-w-[calc(100%-2rem)] gap-3 p-5 sm:max-w-[72rem]">
        <DialogHeader>
          <DialogTitle>Generate Insights</DialogTitle>
          <DialogDescription>
            Run the full insights workflow from here. Site Discovery stays in
            Sources because it expands what you crawl, not how insights are
            generated.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-1">
          <div className="rounded-xl border border-white/[0.08] bg-black/30 p-4">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Badge variant="outline">Primary workflow</Badge>
                  <WorkflowStatusBadge
                    status={pipelineStatus}
                    fallbackLabel="Ready"
                  />
                </div>
                <div>
                  <h3 className="text-base font-semibold">Run Full Pipeline</h3>
                  <p className="text-sm text-muted-foreground">
                    Crawl your tracked sources first, then generate insights
                    from the newly crawled content.
                  </p>
                </div>
                {pipelineStatus && (
                  <div className="space-y-1.5">
                    {pipelineStatus.state === "running" && (
                      <Progress
                        value={pipelineStatus.progress}
                        className="h-2 max-w-xl"
                      />
                    )}
                    <p className="text-sm text-muted-foreground">
                      {getWorkflowMessage(pipelineStatus)}
                    </p>
                  </div>
                )}
              </div>
              <div className="flex items-center gap-2">
                {pipelineRunning ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onStop(PIPELINE_WORKER_NAME)}
                    disabled={activeAction === `stop:${PIPELINE_WORKER_NAME}`}
                  >
                    {activeAction === `stop:${PIPELINE_WORKER_NAME}` ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <>
                        <Square className="w-4 h-4 mr-2" />
                        Stop Pipeline
                      </>
                    )}
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    onClick={() => onRun(PIPELINE_WORKER_NAME)}
                    disabled={
                      workflowBusy ||
                      activeAction === `run:${PIPELINE_WORKER_NAME}`
                    }
                  >
                    {activeAction === `run:${PIPELINE_WORKER_NAME}` ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Starting
                      </>
                    ) : (
                      <>
                        <WandSparkles className="w-4 h-4 mr-2" />
                        Run Full Pipeline
                      </>
                    )}
                  </Button>
                )}
              </div>
            </div>
          </div>

          <div className="grid gap-3 lg:grid-cols-2">
            <WorkflowStepCard
              workerName="crawl"
              status={crawlStatus}
              activeAction={activeAction}
              workflowBusy={workflowBusy}
              onRun={onRun}
              onStop={onStop}
            />
            <WorkflowStepCard
              workerName="insight"
              status={insightStatus}
              activeAction={activeAction}
              workflowBusy={workflowBusy}
              onRun={onRun}
              onStop={onStop}
            />
          </div>

          <div className="rounded-xl border border-dashed border-white/[0.08] p-3 text-sm text-muted-foreground">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="font-medium text-foreground">
                  Source management and Site Discovery
                </p>
                <p className="mt-1">
                  Manage your tracked inputs and discover related sites from the
                  Sources page.
                </p>
              </div>
              <Link to="/dashboard/insights/sources">
                <Button variant="outline" size="sm">
                  Open Sources
                  <ArrowRight className="w-4 h-4 ml-2" />
                </Button>
              </Link>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function WorkflowStepCard({
  workerName,
  status,
  activeAction,
  workflowBusy,
  onRun,
  onStop,
}: {
  workerName: string;
  status?: WorkerStatus;
  activeAction: string | null;
  workflowBusy: boolean | undefined;
  onRun: (workerName: string) => void;
  onStop: (workerName: string) => void;
}) {
  const isRunning = status?.state === "running";
  const runActionKey = `run:${workerName}`;
  const stopActionKey = `stop:${workerName}`;

  return (
    <Card className="gap-4 border-white/[0.08] py-4">
      <CardHeader className="space-y-2 px-4">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-base">
                {getWorkerDisplayName(workerName)}
              </CardTitle>
              <WorkflowStatusBadge status={status} fallbackLabel="Ready" />
            </div>
            <p className="text-sm text-muted-foreground">
              {getWorkerDescription(workerName)}
            </p>
          </div>
          {isRunning ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onStop(workerName)}
              disabled={activeAction === stopActionKey}
            >
              {activeAction === stopActionKey ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <>
                  <Square className="w-4 h-4 mr-2" />
                  Stop
                </>
              )}
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onRun(workerName)}
              disabled={Boolean(workflowBusy) || activeAction === runActionKey}
            >
              {activeAction === runActionKey ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <>
                  <Play className="w-4 h-4 mr-2" />
                  Run
                </>
              )}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-2 px-4">
        {isRunning && <Progress value={status.progress} className="h-2" />}
        <p className="text-sm text-muted-foreground">
          {getWorkflowMessage(status)}
        </p>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Clock3 className="w-3.5 h-3.5" />
          <span>
            {status?.completed_at
              ? `Last completed ${new Date(status.completed_at).toLocaleString()}`
              : "No completed run yet"}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function WorkflowStatusBadge({
  status,
  fallbackLabel,
}: {
  status?: WorkerStatus;
  fallbackLabel: string;
}) {
  const state = status?.state ?? "idle";
  const label = status ? getWorkerStateLabel(status.state) : fallbackLabel;

  return (
    <Badge variant={getWorkerBadgeVariant(state)} className="capitalize">
      {label}
    </Badge>
  );
}

function getWorkerBadgeVariant(state: WorkerState | "idle") {
  switch (state) {
    case "completed":
      return "default" as const;
    case "failed":
      return "destructive" as const;
    case "running":
      return "secondary" as const;
    default:
      return "outline" as const;
  }
}

function getWorkerStateLabel(state: WorkerState) {
  switch (state) {
    case "running":
      return "Running";
    case "completed":
      return "Completed";
    case "failed":
      return "Failed";
    default:
      return "Idle";
  }
}

function getWorkflowMessage(status?: WorkerStatus) {
  if (!status) {
    return "Ready to run.";
  }

  if (status.state === "failed") {
    return status.error || status.message || "This run failed.";
  }

  if (status.state === "completed") {
    return status.message || "Completed successfully.";
  }

  if (status.state === "running") {
    return status.message || "Processing...";
  }

  return "Ready to run.";
}

interface InsightCardProps {
  insight: Insight;
  onMarkAsRead: (id: string) => void;
  onTogglePin: (e: React.MouseEvent, id: string) => void;
  formatDate: (date: string) => string;
}

function InsightCard({
  insight,
  onMarkAsRead,
  onTogglePin,
  formatDate,
}: InsightCardProps) {
  return (
    <Link
      to={`/dashboard/insights/${insight.id}`}
      onClick={() => !insight.is_read && onMarkAsRead(insight.id)}
    >
      <Card
        className={`cursor-pointer transition-all hover:border-primary/50 ${!insight.is_read ? "border-l-4 border-l-primary" : ""}`}
      >
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
                <Pin
                  className={`w-3 h-3 ${insight.is_pinned ? "fill-current" : ""}`}
                />
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
              <Badge variant="default" className="text-xs">
                New
              </Badge>
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
                <div
                  key={i}
                  className="flex items-start gap-2 text-xs text-muted-foreground"
                >
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
            {insight.source_content_ids &&
              insight.source_content_ids.length > 0 && (
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
