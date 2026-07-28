import { apiGet } from "./authenticatedFetch";

export type TaskRunStatus =
  | "queued"
  | "running"
  | "completed"
  | "warning"
  | "failed"
  | "cancelled";

export interface TaskRun {
  id: string;
  kind: string;
  task_name: string;
  status: TaskRunStatus;
  trigger_source: string;
  summary?: string;
  error_summary?: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  output_summary?: Record<string, unknown>;
  metrics?: Record<string, unknown>;
  parent_run_id?: string;
}

export interface TaskRunStep {
  id: string;
  step_key: string;
  step_name: string;
  status: TaskRunStatus;
  summary?: string;
  error_summary?: string;
  started_at?: string;
  completed_at?: string;
  metrics?: Record<string, unknown>;
}

export interface TaskRunEvent {
  id: string;
  event_type: string;
  level: string;
  message: string;
  created_at: string;
  step_key?: string;
  meta_data?: Record<string, unknown>;
}

export interface TaskRunListResponse {
  runs: TaskRun[];
}

export interface TaskRunDetailResponse {
  run: TaskRun;
  steps: TaskRunStep[];
  events: TaskRunEvent[];
}

interface ListTaskRunsParams {
  taskName?: string;
  status?: TaskRunStatus | "all";
  kind?: string;
  limit?: number;
}

export async function listTaskRuns(params: ListTaskRunsParams = {}): Promise<TaskRunListResponse> {
  const search = new URLSearchParams();
  if (params.taskName) {
    search.set("task_name", params.taskName);
  }
  if (params.status && params.status !== "all") {
    search.set("status", params.status);
  }
  if (params.kind) {
    search.set("kind", params.kind);
  }
  if (params.limit) {
    search.set("limit", String(params.limit));
  }
  const query = search.toString();
  return apiGet<TaskRunListResponse>(`/task-runs${query ? `?${query}` : ""}`);
}

export async function getTaskRun(taskRunId: string): Promise<TaskRunDetailResponse> {
  return apiGet<TaskRunDetailResponse>(`/task-runs/${taskRunId}`);
}

export function getTaskRunStatusLabel(status: TaskRunStatus): string {
  switch (status) {
    case "warning":
      return "Needs attention";
    case "failed":
      return "Failed";
    case "cancelled":
      return "Cancelled";
    case "running":
      return "Running";
    case "queued":
      return "Queued";
    default:
      return "Completed";
  }
}
