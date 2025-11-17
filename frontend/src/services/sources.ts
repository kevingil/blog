import { VITE_API_BASE_URL } from './constants';
import { handleApiResponse } from './apiHelpers';
import type { 
  ArticleSource, 
  CreateSourceRequest, 
  UpdateSourceRequest, 
  ScrapeSourceRequest 
} from './types';

// Get all sources for an article
export async function getArticleSources(articleId: string): Promise<ArticleSource[]> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/article/${articleId}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const data = await handleApiResponse<{ sources: ArticleSource[] }>(response);
  return data.sources || [];
}

// Create a new source
export async function createSource(request: CreateSourceRequest): Promise<ArticleSource> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  return handleApiResponse<ArticleSource>(response);
}

// Scrape a URL and create a source
export async function scrapeAndCreateSource(request: ScrapeSourceRequest): Promise<ArticleSource> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/scrape`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  return handleApiResponse<ArticleSource>(response);
}

// Get a specific source
export async function getSource(sourceId: string): Promise<ArticleSource> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/${sourceId}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  return handleApiResponse<ArticleSource>(response);
}

// Update a source
export async function updateSource(sourceId: string, request: UpdateSourceRequest): Promise<ArticleSource> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/${sourceId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  return handleApiResponse<ArticleSource>(response);
}

// Delete a source
export async function deleteSource(sourceId: string): Promise<void> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/${sourceId}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  await handleApiResponse<{ success: boolean }>(response);
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

  const response = await fetch(`${VITE_API_BASE_URL}/sources/article/${articleId}/search?${params}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  const data = await handleApiResponse<{ sources: ArticleSource[] }>(response);
  return data.sources || [];
}
