import { TaskRuns } from "@/client";
import { generatedData } from "./generatedClient";

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
  const query = {
    task_name: params.taskName,
    status: params.status === "all" ? undefined : params.status,
    kind: params.kind,
    limit: params.limit === undefined ? undefined : String(params.limit),
  };

  return generatedData<TaskRunListResponse>(TaskRuns.listTaskRuns({ query }));
}

export async function getTaskRun(taskRunId: string): Promise<TaskRunDetailResponse> {
  return generatedData<TaskRunDetailResponse>(
    TaskRuns.getTaskRun({ path: { id: taskRunId } }),
  );
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
