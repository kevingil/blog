import { useEffect } from "react";
import { Link, createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Clock3 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import { getTaskRun, getTaskRunStatusLabel } from "@/services/taskRuns";

export const Route = createFileRoute("/dashboard/tasks/$taskRunId")({
  component: TaskRunDetailPage,
});

function TaskRunDetailPage() {
  const { taskRunId } = Route.useParams();
  const { setPageTitle } = useAdminDashboard();

  const { data, isLoading } = useQuery({
    queryKey: ["task-run", taskRunId],
    queryFn: () => getTaskRun(taskRunId),
  });

  useEffect(() => {
    setPageTitle("Task Details");
  }, [setPageTitle]);

  if (isLoading) {
    return <section className="flex-1 overflow-auto p-6 text-sm text-muted-foreground">Loading task details...</section>;
  }

  if (!data) {
    return <section className="flex-1 overflow-auto p-6 text-sm text-muted-foreground">Task run not found.</section>;
  }

  const { run, steps, events } = data;

  return (
    <section className="flex-1 overflow-auto p-6 space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <Link to="/dashboard/tasks">
            <Button variant="ghost" size="sm" className="px-0">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Tasks
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-semibold">{run.task_name}</h1>
            <p className="text-sm text-muted-foreground">
              {run.summary || run.error_summary || "No summary provided"}
            </p>
          </div>
        </div>
        <Badge variant={run.status === "failed" ? "destructive" : run.status === "warning" ? "secondary" : "outline"}>
          {getTaskRunStatusLabel(run.status)}
        </Badge>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <Card className="gap-4 py-4">
          <CardHeader className="px-5">
            <CardTitle className="text-base">Summary</CardTitle>
          </CardHeader>
          <CardContent className="px-5 space-y-3 text-sm">
            <div className="grid gap-3 md:grid-cols-2">
              <DetailItem label="Task" value={run.task_name} />
              <DetailItem label="Kind" value={run.kind} />
              <DetailItem label="Trigger" value={run.trigger_source} />
              <DetailItem label="Started" value={run.started_at ? new Date(run.started_at).toLocaleString() : "Not started"} />
            </div>
            {run.metrics && Object.keys(run.metrics).length > 0 && (
              <pre className="rounded-lg border border-white/[0.08] bg-black/30 p-3 text-xs whitespace-pre-wrap">
                {JSON.stringify(run.metrics, null, 2)}
              </pre>
            )}
          </CardContent>
        </Card>

        <Card className="gap-4 py-4">
          <CardHeader className="px-5">
            <CardTitle className="text-base">Output</CardTitle>
          </CardHeader>
          <CardContent className="px-5 text-sm">
            {run.output_summary && Object.keys(run.output_summary).length > 0 ? (
              <pre className="rounded-lg border border-white/[0.08] bg-black/30 p-3 text-xs whitespace-pre-wrap">
                {JSON.stringify(run.output_summary, null, 2)}
              </pre>
            ) : (
              <p className="text-muted-foreground">No output summary recorded.</p>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <Card className="gap-4 py-4">
          <CardHeader className="px-5">
            <CardTitle className="text-base">Steps</CardTitle>
          </CardHeader>
          <CardContent className="px-5 space-y-3">
            {steps.length === 0 ? (
              <p className="text-sm text-muted-foreground">No step data recorded.</p>
            ) : (
              steps.map((step) => (
                <div key={step.id} className="rounded-lg border border-white/[0.08] p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="font-medium">{step.step_name}</div>
                      <div className="text-xs text-muted-foreground">{step.summary || step.error_summary || step.step_key}</div>
                    </div>
                    <Badge variant={step.status === "failed" ? "destructive" : step.status === "warning" ? "secondary" : "outline"}>
                      {getTaskRunStatusLabel(step.status)}
                    </Badge>
                  </div>
                </div>
              ))
            )}
          </CardContent>
        </Card>

        <Card className="gap-4 py-4">
          <CardHeader className="px-5">
            <CardTitle className="text-base">Events</CardTitle>
          </CardHeader>
          <CardContent className="px-5 space-y-3">
            {events.length === 0 ? (
              <p className="text-sm text-muted-foreground">No events recorded.</p>
            ) : (
              events.map((event) => (
                <div key={event.id} className="rounded-lg border border-white/[0.08] p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium text-sm">{event.message}</div>
                    <Badge variant={event.level === "error" ? "destructive" : event.level === "warning" ? "secondary" : "outline"}>
                      {event.level}
                    </Badge>
                  </div>
                  <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <Clock3 className="w-3.5 h-3.5" />
                    <span>{new Date(event.created_at).toLocaleString()}</span>
                    {event.step_key ? <span>• {event.step_key}</span> : null}
                  </div>
                  {event.meta_data && Object.keys(event.meta_data).length > 0 ? (
                    <pre className="mt-3 rounded-md border border-white/[0.08] bg-black/30 p-2 text-[11px] whitespace-pre-wrap">
                      {JSON.stringify(event.meta_data, null, 2)}
                    </pre>
                  ) : null}
                </div>
              ))
            )}
          </CardContent>
        </Card>
      </div>
    </section>
  );
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-white/[0.08] p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1">{value}</div>
    </div>
  );
}
