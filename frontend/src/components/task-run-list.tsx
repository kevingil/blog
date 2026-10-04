import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, RefreshCw } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getTaskRunStatusLabel,
  listTaskRuns,
  type TaskRun,
  type TaskRunStatus,
} from "@/services/taskRuns";

export function TaskRunList() {
  const [statusFilter, setStatusFilter] = useState<TaskRunStatus | "all">("all");
  const [taskNameFilter, setTaskNameFilter] = useState("all");

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ["task-runs", statusFilter, taskNameFilter],
    queryFn: () =>
      listTaskRuns({
        status: statusFilter,
        taskName: taskNameFilter === "all" ? undefined : taskNameFilter,
        limit: 100,
      }),
  });

  const runs = data?.runs ?? [];

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <p className="text-sm text-muted-foreground max-w-xl">
          Check now and the scheduled insight check are recorded here. Crawl and site discovery are still registered workers, but the schedule only runs insight.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <Select
            value={statusFilter}
            onValueChange={(value) => setStatusFilter(value as TaskRunStatus | "all")}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="running">Running</SelectItem>
              <SelectItem value="warning">Needs attention</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
            </SelectContent>
          </Select>
          <Select value={taskNameFilter} onValueChange={setTaskNameFilter}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Filter run" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All runs</SelectItem>
              <SelectItem value="insight">Insight Generator</SelectItem>
              <SelectItem value="pipeline">Full Pipeline</SelectItem>
              <SelectItem value="crawl">Content Crawler</SelectItem>
              <SelectItem value="discovery">Site Discovery</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
            <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      <Card className="gap-0 py-0">
        <CardHeader className="px-6 py-4 border-b border-border">
          <CardTitle className="text-base">Recent runs</CardTitle>
        </CardHeader>
        <CardContent className="px-0">
          {isLoading ? (
            <div className="px-6 py-10 text-sm text-muted-foreground">Loading runs...</div>
          ) : isError ? (
            <div className="px-6 py-10 text-sm text-destructive">
              Runs could not be loaded.
            </div>
          ) : runs.length === 0 ? (
            <div className="px-6 py-10 text-sm text-muted-foreground">
              No runs recorded yet. A Check now from What to watch writes one.
            </div>
          ) : (
            <Table noWrapper>
              <TableHeader>
                <TableRow>
                  <TableHead className="px-6">Run</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Kind</TableHead>
                  <TableHead>Started</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Summary</TableHead>
                  <TableHead className="w-[100px] px-6">Open</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.id}>
                    <TableCell className="px-6">
                      <div className="font-medium">{taskNameLabel(run)}</div>
                      <div className="text-xs text-muted-foreground">{run.trigger_source}</div>
                    </TableCell>
                    <TableCell>
                      <TaskRunStatusBadge status={run.status} />
                    </TableCell>
                    <TableCell className="capitalize text-muted-foreground">{run.kind}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDateTime(run.started_at)}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDuration(run.duration_ms)}
                    </TableCell>
                    <TableCell className="max-w-[420px] whitespace-normal text-sm text-muted-foreground">
                      {run.summary || run.error_summary || "No summary provided"}
                    </TableCell>
                    <TableCell className="px-6">
                      <Link to="/dashboard/tasks/$taskRunId" params={{ taskRunId: run.id }}>
                        <Button variant="ghost" size="sm">
                          <ExternalLink className="w-4 h-4 mr-2" />
                          View
                        </Button>
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function TaskRunStatusBadge({ status }: { status: TaskRun["status"] }) {
  const variant =
    status === "failed"
      ? "destructive"
      : status === "warning"
        ? "secondary"
        : status === "running"
          ? "default"
          : "outline";

  return <Badge variant={variant}>{getTaskRunStatusLabel(status)}</Badge>;
}

function taskNameLabel(run: TaskRun) {
  switch (run.task_name) {
    case "pipeline":
      return "Full Pipeline";
    case "crawl":
      return "Content Crawler";
    case "insight":
      return "Insight Generator";
    case "discovery":
      return "Site Discovery";
    default:
      return run.task_name;
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return "Not started";
  }
  return new Date(value).toLocaleString();
}

function formatDuration(durationMS?: number) {
  if (!durationMS && durationMS !== 0) {
    return "In progress";
  }
  const seconds = Math.round(durationMS / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
}
