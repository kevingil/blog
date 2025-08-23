import { VITE_API_BASE_URL } from './constants';
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

  if (!response.ok) {
    throw new Error(`Failed to fetch sources: ${response.statusText}`);
  }

  const data = await response.json();
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

  if (!response.ok) {
    throw new Error(`Failed to create source: ${response.statusText}`);
  }

  return response.json();
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

  if (!response.ok) {
    throw new Error(`Failed to scrape and create source: ${response.statusText}`);
  }

  return response.json();
}

// Get a specific source
export async function getSource(sourceId: string): Promise<ArticleSource> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/${sourceId}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch source: ${response.statusText}`);
  }

  return response.json();
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

  if (!response.ok) {
    throw new Error(`Failed to update source: ${response.statusText}`);
  }

  return response.json();
}

// Delete a source
export async function deleteSource(sourceId: string): Promise<void> {
  const response = await fetch(`${VITE_API_BASE_URL}/sources/${sourceId}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error(`Failed to delete source: ${response.statusText}`);
  }
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

  if (!response.ok) {
    throw new Error(`Failed to search sources: ${response.statusText}`);
  }

  const data = await response.json();
  return data.sources || [];
}
