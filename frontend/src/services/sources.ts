import { Sources } from '@/client';
import { generatedData } from './generatedClient';
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
  return generatedData<GetAllSourcesResponse>(
    Sources.listAllSources({ query: { page, limit } }),
  );
}

// Get all sources for an article
export async function getArticleSources(articleId: string): Promise<ArticleSource[]> {
  const data = await generatedData<{ sources: ArticleSource[] }>(
    Sources.getArticleSources({ path: { articleId } }),
  );
  return data.sources || [];
}

// Create a new source
export async function createSource(request: CreateSourceRequest): Promise<ArticleSource> {
  return generatedData<ArticleSource>(Sources.createSource({ body: request }));
}

// Scrape a URL and create a source
export async function scrapeAndCreateSource(request: ScrapeSourceRequest): Promise<ArticleSource> {
  return generatedData<ArticleSource>(Sources.scrapeAndCreateSource({ body: request }));
}

// Get a specific source
export async function getSource(sourceId: string): Promise<ArticleSource> {
  return generatedData<ArticleSource>(Sources.getSource({ path: { sourceId } }));
}

// Update a source
export async function updateSource(sourceId: string, request: UpdateSourceRequest): Promise<ArticleSource> {
  return generatedData<ArticleSource>(
    Sources.updateSource({ path: { sourceId }, body: request }),
  );
}

// Delete a source
export async function deleteSource(sourceId: string): Promise<void> {
  await generatedData<{ success: boolean }>(Sources.deleteSource({ path: { sourceId } }));
}

// Search for similar sources
export async function searchSimilarSources(
  articleId: string, 
  query: string, 
  limit: number = 5
): Promise<ArticleSource[]> {
  const data = await generatedData<{ sources: ArticleSource[] }>(
    Sources.searchSimilarSources({ path: { articleId }, query: { q: query, limit } }),
  );
  return data.sources || [];
}
