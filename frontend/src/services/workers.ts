import { apiGet, apiPost } from './authenticatedFetch';

// Types
export type WorkerState = 'idle' | 'running' | 'completed' | 'failed';

export interface WorkerStatus {
  name: string;
  state: WorkerState;
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
}

export interface StopWorkerResponse {
  stopped: boolean;
  message: string;
}

export interface RunningWorkersResponse {
  workers: string[];
}

// API calls

export async function getWorkersStatus(): Promise<AllWorkersStatusResponse> {
  return apiGet<AllWorkersStatusResponse>('/workers/status');
}

export async function getWorkerStatus(name: string): Promise<WorkerStatus> {
  return apiGet<WorkerStatus>(`/workers/${name}/status`);
}

export async function runWorker(name: string): Promise<RunWorkerResponse> {
  return apiPost<RunWorkerResponse>(`/workers/${name}/run`);
}

export async function stopWorker(name: string): Promise<StopWorkerResponse> {
  return apiPost<StopWorkerResponse>(`/workers/${name}/stop`);
}

export async function getRunningWorkers(): Promise<RunningWorkersResponse> {
  return apiGet<RunningWorkersResponse>('/workers/running');
}

// Worker display names
export const WORKER_DISPLAY_NAMES: Record<string, string> = {
  crawl: 'Content Crawler',
  insight: 'Insight Generator',
  discovery: 'Site Discovery',
};

// Worker descriptions
export const WORKER_DESCRIPTIONS: Record<string, string> = {
  crawl: 'Crawls configured data sources and extracts content',
  insight: 'Generates AI-powered insights from crawled content',
  discovery: 'Discovers similar websites using Exa search',
};

// Get display name for a worker
export function getWorkerDisplayName(name: string): string {
  return WORKER_DISPLAY_NAMES[name] || name;
}

// Get description for a worker
export function getWorkerDescription(name: string): string {
  return WORKER_DESCRIPTIONS[name] || '';
}
