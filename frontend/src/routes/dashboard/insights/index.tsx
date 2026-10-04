import { useEffect, useRef, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Lightbulb,
  Tag,
  Calendar,
  Pin,
  Check,
  Loader2,
  Play,
  Search,
  Trash2,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { ResearchImage } from "./research-image";
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import {
  checkTracker,
  createTracker,
  deleteTracker,
  listInsights,
  listTopics,
  listTrackers,
  markInsightAsRead,
  searchInsights,
  toggleInsightPinned,
  updateTracker,
  type Insight,
  type Tracker,
} from "@/services/insights";
import { useWorkerStatuses } from "@/hooks/use-worker-statuses";
import { getWorkerDisplayName, type WorkerStatus } from "@/services/workers";

export const Route = createFileRoute("/dashboard/insights/")({
  component: InsightsPage,
});

const FREQUENCIES = [
  { value: "hourly", label: "Hourly" },
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
];

function InsightsPage() {
  const [page, setPage] = useState(1);
  const [selectedTopicId, setSelectedTopicId] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();
  const workerStatuses = useWorkerStatuses();
  const previousInsightStatus = useRef<WorkerStatus | undefined>(undefined);

  useEffect(() => {
    setPageTitle("Insights");
  }, [setPageTitle]);

  useEffect(() => {
    const nextStatus = workerStatuses.insight;
    const previousStatus = previousInsightStatus.current;
    previousInsightStatus.current = nextStatus;
    if (!nextStatus || previousStatus?.state === nextStatus.state) {
      return;
    }
    if (nextStatus.state === "completed") {
      toast({
        title: `${getWorkerDisplayName("insight")} completed`,
        description: nextStatus.message || "The research check finished.",
      });
      queryClient.invalidateQueries({ queryKey: ["insights"] });
      queryClient.invalidateQueries({ queryKey: ["insight-trackers"] });
    }
    if (nextStatus.state === "failed") {
      toast({
        title: `${getWorkerDisplayName("insight")} failed`,
        description: nextStatus.error || nextStatus.message || "The research check failed.",
        variant: "destructive",
      });
    }
  }, [queryClient, toast, workerStatuses.insight]);

  const { data: topics = [] } = useQuery({
    queryKey: ["insight-topics"],
    queryFn: listTopics,
  });

  const { data: insightsData, isLoading } = useQuery({
    queryKey: ["insights", page, selectedTopicId],
    queryFn: () =>
      listInsights(page, 12, selectedTopicId === "all" ? undefined : selectedTopicId),
  });

  const { data: searchResults, isLoading: isSearchLoading } = useQuery({
    queryKey: ["insights-search", searchQuery],
    queryFn: () => searchInsights(searchQuery, 20),
    enabled: searchQuery.length > 2,
  });

  const insights = searchQuery.length > 2 ? searchResults || [] : insightsData?.insights || [];
  const total = insightsData?.total || 0;
  const totalPages = Math.ceil(total / 12);

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

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  };

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto space-y-6">
      <TrackerPanel insightRunning={workerStatuses.insight?.state === "running"} />

      <div className="flex flex-col md:flex-row gap-4">
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
        <Link to="/dashboard/tasks">
          <Button variant="outline">Tasks</Button>
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
          <p className="text-lg font-medium mb-2">No briefings yet</p>
          <p className="text-sm">
            Add a subject or a site above. The next check writes what people are saying and what is worth writing.
          </p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {insights.map((insight) => (
              <InsightCard
                key={insight.id}
                insight={insight}
                onMarkAsRead={(id) => markReadMutation.mutate(id)}
                onTogglePin={(event, id) => {
                  event.stopPropagation();
                  togglePinMutation.mutate(id);
                }}
                formatDate={formatDate}
              />
            ))}
          </div>

          {totalPages > 1 && searchQuery.length <= 2 && (
            <div className="mt-6 flex justify-center">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => setPage(Math.max(1, page - 1))}
                      className={
                        page === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"
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
    </section>
  );
}

function TrackerPanel({ insightRunning }: { insightRunning: boolean }) {
  const [target, setTarget] = useState("");
  const [frequency, setFrequency] = useState("daily");
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const { data: trackers = [], isLoading } = useQuery({
    queryKey: ["insight-trackers"],
    queryFn: listTrackers,
  });

  const createMutation = useMutation({
    mutationFn: () => createTracker({ target: target.trim(), frequency }),
    onSuccess: (tracker) => {
      setTarget("");
      queryClient.invalidateQueries({ queryKey: ["insight-trackers"] });
      toast({
        title: "Tracker added",
        description: `${tracker.name} will be checked ${tracker.frequency}.`,
      });
    },
    onError: (error) => {
      toast({
        title: "Could not add tracker",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    },
  });

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">What to watch</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <form
          className="flex flex-col gap-3 md:flex-row md:items-end"
          onSubmit={(event) => {
            event.preventDefault();
            if (target.trim()) {
              createMutation.mutate();
            }
          }}
        >
          <div className="flex-1 space-y-1.5">
            <Label htmlFor="tracker-target">Subject or site</Label>
            <Input
              id="tracker-target"
              value={target}
              onChange={(event) => setTarget(event.target.value)}
              placeholder="pgvector or https://blog.rust-lang.org"
            />
          </div>
          <div className="space-y-1.5">
            <Label>How often</Label>
            <Select value={frequency} onValueChange={setFrequency}>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FREQUENCIES.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button type="submit" disabled={!target.trim() || createMutation.isPending}>
            {createMutation.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              "Track"
            )}
          </Button>
        </form>

        {isLoading ? (
          <div className="flex items-center text-sm text-muted-foreground">
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            Loading trackers...
          </div>
        ) : trackers.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Nothing is being watched yet. A subject tracks the conversation. A full http(s) URL tracks that site.
          </p>
        ) : (
          <ul className="divide-y rounded-md border">
            {trackers.map((tracker) => (
              <TrackerRow key={`${tracker.kind}-${tracker.id}`} tracker={tracker} busy={insightRunning} />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function TrackerRow({ tracker, busy }: { tracker: Tracker; busy: boolean }) {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["insight-trackers"] });
  };
  const updateMutation = useMutation({
    mutationFn: (frequency: string) => updateTracker(tracker.kind, tracker.id, { frequency }),
    onSuccess: refresh,
    onError: (error) => {
      toast({
        title: "Could not update tracker",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    },
  });
  const deleteMutation = useMutation({
    mutationFn: () => deleteTracker(tracker.kind, tracker.id),
    onSuccess: refresh,
    onError: (error) => {
      toast({
        title: "Could not remove tracker",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    },
  });
  const checkMutation = useMutation({
    mutationFn: () => checkTracker(tracker.kind, tracker.id),
    onSuccess: (result) => {
      refresh();
      toast({
        title: result.started ? "Check started" : "Check queued",
        description: result.started
          ? `${tracker.name} is being researched now.`
          : `${tracker.name} is due and will run when the current check finishes.`,
      });
    },
    onError: (error) => {
      toast({
        title: "Could not start check",
        description: error instanceof Error ? error.message : "Unknown error",
        variant: "destructive",
      });
    },
  });

  return (
    <li className="flex flex-col gap-3 p-3 md:flex-row md:items-center">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium truncate">{tracker.name}</span>
          <Badge variant="outline">{tracker.kind === "domain" ? "Site" : "Subject"}</Badge>
          {!tracker.enabled && <Badge variant="secondary">Paused</Badge>}
        </div>
        <p className="text-xs text-muted-foreground truncate">{tracker.target}</p>
      </div>
      <Select
        value={tracker.frequency}
        onValueChange={(value) => updateMutation.mutate(value)}
        disabled={updateMutation.isPending}
      >
        <SelectTrigger className="w-[140px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {FREQUENCIES.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button
        variant="outline"
        size="sm"
        onClick={() => checkMutation.mutate()}
        disabled={checkMutation.isPending}
      >
        {checkMutation.isPending || busy ? (
          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
        ) : (
          <Play className="w-4 h-4 mr-2" />
        )}
        Check now
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => deleteMutation.mutate()}
        disabled={deleteMutation.isPending}
        aria-label={`Remove ${tracker.name}`}
      >
        <Trash2 className="w-4 h-4" />
      </Button>
    </li>
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
    <Link
      to="/dashboard/insights/$insightId"
      params={{ insightId: insight.id }}
      onClick={() => !insight.is_read && onMarkAsRead(insight.id)}
    >
      <Card
        className={`cursor-pointer overflow-hidden transition-all hover:border-primary/50 ${!insight.is_read ? "border-l-4 border-l-primary" : ""}`}
      >
        <ResearchImage
          src={insight.image_url}
          className="-mt-6 aspect-video w-full object-cover"
        />
        <CardHeader className="pb-2">
          <div className="flex items-start justify-between gap-2">
            <CardTitle className="text-sm font-medium line-clamp-2 flex-1">{insight.title}</CardTitle>
            <div className="flex items-center gap-1">
              {insight.is_pinned && <Pin className="w-3 h-3 text-primary fill-primary" />}
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={(e) => onTogglePin(e, insight.id)}
              >
                <Pin className={`w-3 h-3 ${insight.is_pinned ? "fill-current" : ""}`} />
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
          <p className="text-sm text-muted-foreground line-clamp-3 mb-3">{insight.summary}</p>
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
