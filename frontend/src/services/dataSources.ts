import { apiGet, apiPost, apiPut, apiDelete } from './authenticatedFetch';
import type { CrawledContent } from './insights';

// Types
export interface DataSource {
  id: string;
  organization_id?: string;
  name: string;
  url: string;
  feed_url?: string;
  source_type: string;
  crawl_frequency: string;
  is_enabled: boolean;
  is_discovered: boolean;
  discovered_from_id?: string;
  last_crawled_at?: string;
  next_crawl_at?: string;
  crawl_status: string;
  error_message?: string;
  content_count: number;
  meta_data?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateDataSourceRequest {
  name: string;
  url: string;
  feed_url?: string;
  source_type?: string;
  crawl_frequency?: string;
  is_enabled?: boolean;
}

export interface UpdateDataSourceRequest {
  name?: string;
  url?: string;
  feed_url?: string;
  source_type?: string;
  crawl_frequency?: string;
  is_enabled?: boolean;
}

// API calls

export interface ListDataSourcesResponse {
  data_sources: DataSource[];
  total: number;
  page: number;
  limit: number;
}

export async function listDataSources(page: number = 1, limit: number = 20): Promise<DataSource[] | ListDataSourcesResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  
  return apiGet<DataSource[] | ListDataSourcesResponse>(`/data-sources?${params}`);
}

export async function getDataSource(id: string): Promise<DataSource> {
  return apiGet<DataSource>(`/data-sources/${id}`);
}

export async function createDataSource(request: CreateDataSourceRequest): Promise<DataSource> {
  return apiPost<DataSource>('/data-sources', request);
}

export async function updateDataSource(id: string, request: UpdateDataSourceRequest): Promise<DataSource> {
  return apiPut<DataSource>(`/data-sources/${id}`, request);
}

export async function deleteDataSource(id: string): Promise<void> {
  await apiDelete<{ success: boolean }>(`/data-sources/${id}`);
}

export async function triggerCrawl(id: string): Promise<void> {
  await apiPost<{ success: boolean; message: string }>(`/data-sources/${id}/crawl`, {});
}

export interface GetDataSourceContentResponse {
  contents: CrawledContent[];
  total: number;
  page: number;
  limit: number;
}

export async function getDataSourceContent(
  id: string, 
  page: number = 1, 
  limit: number = 20
): Promise<GetDataSourceContentResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  
  return apiGet<GetDataSourceContentResponse>(`/data-sources/${id}/content?${params}`);
}
