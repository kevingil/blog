import { useEffect, useState } from "react";
import { Link, createFileRoute } from "@tanstack/react-router";
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
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import {
  getTaskRunStatusLabel,
  listTaskRuns,
  type TaskRun,
  type TaskRunStatus,
} from "@/services/taskRuns";

export const Route = createFileRoute("/dashboard/tasks/")({
  component: TasksPage,
});

function TasksPage() {
  const { setPageTitle } = useAdminDashboard();
  const [statusFilter, setStatusFilter] = useState<TaskRunStatus | "all">("all");
  const [taskNameFilter, setTaskNameFilter] = useState("all");

  useEffect(() => {
    setPageTitle("Tasks");
  }, [setPageTitle]);

  const { data, isLoading, refetch, isFetching } = useQuery({
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
    <section className="flex-1 overflow-auto p-6 space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Tasks</h1>
          <p className="text-sm text-muted-foreground">
            Review durable workflow and agent run history across the product.
          </p>
        </div>
        <div className="flex items-center gap-3">
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
              <SelectValue placeholder="Filter task" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All tasks</SelectItem>
              <SelectItem value="pipeline">Full Pipeline</SelectItem>
              <SelectItem value="crawl">Content Crawler</SelectItem>
              <SelectItem value="insight">Insight Generator</SelectItem>
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
        <CardHeader className="px-6 py-4 border-b border-white/[0.08]">
          <CardTitle className="text-base">Recent Runs</CardTitle>
        </CardHeader>
        <CardContent className="px-0">
          {isLoading ? (
            <div className="px-6 py-10 text-sm text-muted-foreground">Loading task history...</div>
          ) : runs.length === 0 ? (
            <div className="px-6 py-10 text-sm text-muted-foreground">
              No task runs recorded yet.
            </div>
          ) : (
            <Table noWrapper>
              <TableHeader>
                <TableRow>
                  <TableHead className="px-6">Task</TableHead>
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
                      <div className="font-medium">{getTaskNameLabel(run)}</div>
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
    </section>
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

function getTaskNameLabel(run: TaskRun) {
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
