import { apiGet, apiPost, apiPut, apiDelete } from './authenticatedFetch';
import type { 
  ArticleSource, 
  ArticleSourceWithArticle,
  CreateSourceRequest, 
  UpdateSourceRequest, 
  ScrapeSourceRequest 
} from './types';

export interface GetAllSourcesResponse {
  sources: ArticleSourceWithArticle[];
  total_pages: number;
  page: number;
}

// Get all sources with pagination (for dashboard)
export async function getAllSources(page: number = 1, limit: number = 20): Promise<GetAllSourcesResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  
  return apiGet<GetAllSourcesResponse>(`/dashboard/sources?${params}`);
}

// Get all sources for an article
export async function getArticleSources(articleId: string): Promise<ArticleSource[]> {
  const data = await apiGet<{ sources: ArticleSource[] }>(`/sources/article/${articleId}`);
  return data.sources || [];
}

// Create a new source
export async function createSource(request: CreateSourceRequest): Promise<ArticleSource> {
  return apiPost<ArticleSource>('/sources/', request);
}

// Scrape a URL and create a source
export async function scrapeAndCreateSource(request: ScrapeSourceRequest): Promise<ArticleSource> {
  return apiPost<ArticleSource>('/sources/scrape', request);
}

// Get a specific source
export async function getSource(sourceId: string): Promise<ArticleSource> {
  return apiGet<ArticleSource>(`/sources/${sourceId}`);
}

// Update a source
export async function updateSource(sourceId: string, request: UpdateSourceRequest): Promise<ArticleSource> {
  return apiPut<ArticleSource>(`/sources/${sourceId}`, request);
}

// Delete a source
export async function deleteSource(sourceId: string): Promise<void> {
  await apiDelete<{ success: boolean }>(`/sources/${sourceId}`);
}

// Search for similar sources
export async function searchSimilarSources(
  articleId: string, 
  query: string, 
  limit: number = 5
): Promise<ArticleSource[]> {
  const params = new URLSearchParams({
    q: query,
    limit: limit.toString(),
  });

  const data = await apiGet<{ sources: ArticleSource[] }>(`/sources/article/${articleId}/search?${params}`);
  return data.sources || [];
}
