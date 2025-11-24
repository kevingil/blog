import { apiGet, apiPost, apiPut, apiDelete } from './authenticatedFetch';
import type { 
  ArticleSource, 
  CreateSourceRequest, 
  UpdateSourceRequest, 
  ScrapeSourceRequest 
} from './types';

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
