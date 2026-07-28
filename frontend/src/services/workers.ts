import { Workers } from "@/client";
import { generatedData } from "./generatedClient";

// Types
export type WorkerState = "idle" | "running" | "completed" | "failed";

export interface WorkerStatus {
  name: string;
  state: WorkerState;
  task_run_id?: string;
  progress: number;
  message: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
  items_total: number;
  items_done: number;
}

export interface AllWorkersStatusResponse {
  workers: WorkerStatus[];
  is_running: boolean;
}

export interface RunWorkerResponse {
  started: boolean;
  message: string;
  task_run_id?: string;
}

export interface StopWorkerResponse {
  stopped: boolean;
  message: string;
}

export interface RunningWorkersResponse {
  workers: string[];
}

export const PIPELINE_WORKER_NAME = "pipeline";

// API calls

export async function getWorkersStatus(): Promise<AllWorkersStatusResponse> {
  return generatedData<AllWorkersStatusResponse>(Workers.getAllWorkerStatus());
}

export async function getWorkerStatus(name: string): Promise<WorkerStatus> {
  return generatedData<WorkerStatus>(Workers.getWorkerStatus({ path: { name } }));
}

export async function runWorker(name: string): Promise<RunWorkerResponse> {
  return generatedData<RunWorkerResponse>(Workers.runWorker({ path: { name } }));
}

export async function stopWorker(name: string): Promise<StopWorkerResponse> {
  return generatedData<StopWorkerResponse>(Workers.stopWorker({ path: { name } }));
}

export async function getRunningWorkers(): Promise<RunningWorkersResponse> {
  return generatedData<RunningWorkersResponse>(Workers.getRunningWorkers());
}

// Worker display names
export const WORKER_DISPLAY_NAMES: Record<string, string> = {
  pipeline: "Full Pipeline",
  crawl: "Content Crawler",
  insight: "Insight Generator",
  discovery: "Site Discovery",
};

// Worker descriptions
export const WORKER_DESCRIPTIONS: Record<string, string> = {
  pipeline: "Runs source crawl and insight generation in sequence",
  crawl: "Crawls configured data sources and extracts content",
  insight: "Generates AI-powered insights from crawled content",
  discovery: "Discovers similar websites using Exa search",
};

// Get display name for a worker
export function getWorkerDisplayName(name: string): string {
  return WORKER_DISPLAY_NAMES[name] || name;
}

// Get description for a worker
export function getWorkerDescription(name: string): string {
  return WORKER_DESCRIPTIONS[name] || "";
}
