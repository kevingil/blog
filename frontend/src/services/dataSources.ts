import { DataSources } from "@/client";
import { generatedData } from "./generatedClient";
import type { CrawledContent } from "./insights";

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

export interface DataSourceRecommendation {
  name: string;
  url: string;
  domain: string;
  summary?: string;
  reason?: string;
  source_type: string;
  score?: number;
  favicon?: string;
  sample_url?: string;
  sample_title?: string;
}

export interface RecommendDataSourcesRequest {
  query: string;
  limit?: number;
}

export interface DiscoverDataSourcesRequest {
  limit?: number;
}

export interface RecommendDataSourcesResponse {
  mode?: "query" | "discovery";
  query: string;
  seed_count?: number;
  recommendations: DataSourceRecommendation[];
}

// API calls

export interface ListDataSourcesResponse {
  data_sources: DataSource[];
  total: number;
  page: number;
  limit: number;
}

export async function listDataSources(
  page: number = 1,
  limit: number = 20,
): Promise<DataSource[] | ListDataSourcesResponse> {
  return generatedData<DataSource[] | ListDataSourcesResponse>(
    DataSources.listDataSources({ query: { page, limit } }),
  );
}

export async function getDataSource(id: string): Promise<DataSource> {
  return generatedData<DataSource>(DataSources.getDataSource({ path: { id } }));
}

export async function createDataSource(
  request: CreateDataSourceRequest,
): Promise<DataSource> {
  return generatedData<DataSource>(DataSources.createDataSource({ body: request }));
}

export async function recommendDataSources(
  request: RecommendDataSourcesRequest,
): Promise<RecommendDataSourcesResponse> {
  return generatedData<RecommendDataSourcesResponse>(
    DataSources.recommendDataSources({ body: request }),
  );
}

export async function discoverDataSourcesFromExistingSources(
  request: DiscoverDataSourcesRequest = {},
): Promise<RecommendDataSourcesResponse> {
  return generatedData<RecommendDataSourcesResponse>(
    DataSources.discoverDataSources({ body: request }),
  );
}

export async function updateDataSource(
  id: string,
  request: UpdateDataSourceRequest,
): Promise<DataSource> {
  return generatedData<DataSource>(
    DataSources.updateDataSource({ path: { id }, body: request }),
  );
}

export async function deleteDataSource(id: string): Promise<void> {
  await generatedData<{ success: boolean }>(
    DataSources.deleteDataSource({ path: { id } }),
  );
}

export async function triggerCrawl(id: string): Promise<void> {
  await generatedData<{ success: boolean; message: string }>(
    DataSources.triggerDataSourceCrawl({ path: { id } }),
  );
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
  limit: number = 20,
): Promise<GetDataSourceContentResponse> {
  return generatedData<GetDataSourceContentResponse>(
    DataSources.getDataSourceContent({ path: { id }, query: { page, limit } }),
  );
}
