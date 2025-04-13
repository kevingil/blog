import { ArticleListItem } from '@/services/types';

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

// Article listing and search
export async function getArticles(page: number, tag: string | null = null): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const params = new URLSearchParams({
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {})
  });

  const response = await fetch(`${API_BASE_URL}/api/blog/articles?${params}`);
  if (!response.ok) {
    throw new Error('Failed to fetch articles');
  }
  return response.json();
}

export async function searchArticles(query: string, page: number = 1, tag: string | null = null): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const params = new URLSearchParams({
    query,
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {})
  });

  const response = await fetch(`${API_BASE_URL}/api/blog/articles/search?${params}`);
  if (!response.ok) {
    throw new Error('Failed to search articles');
  }
  return response.json();
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  const response = await fetch(`${API_BASE_URL}/api/blog/tags/popular`);
  if (!response.ok) {
    throw new Error('Failed to fetch popular tags');
  }
  return response.json();
}

// Article CRUD operations
export async function getArticle(slug: string): Promise<ArticleListItem | null> {
  const response = await fetch(`${API_BASE_URL}/api/blog/articles/${slug}`);
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error('Failed to fetch article');
  }
  return response.json();
}

export async function createArticle(article: {
  title: string;
  content: string;
  image?: string;
  tags: string[];
  isDraft: boolean;
  authorId: number;
}): Promise<ArticleListItem> {
  const response = await fetch(`${API_BASE_URL}/api/blog/articles`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(article),
  });
  if (!response.ok) {
    throw new Error('Failed to create article');
  }
  return response.json();
}

export async function updateArticle(slug: string, article: {
  title: string;
  content: string;
  image?: string;
  tags: string[];
  isDraft: boolean;
  publishedAt: number;
}): Promise<ArticleListItem> {
  const response = await fetch(`${API_BASE_URL}/api/blog/articles/${slug}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(article),
  });
  if (!response.ok) {
    throw new Error('Failed to update article');
  }
  return response.json();
}

// Article image operations
export async function generateArticleImage(prompt: string, articleId: number): Promise<{ success: boolean; generationRequestId: string }> {
  const response = await fetch(`${API_BASE_URL}/api/blog/articles/${articleId}/image`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ prompt }),
  });
  if (!response.ok) {
    throw new Error('Failed to generate article image');
  }
  return response.json();
}

export async function getImageGeneration(requestId: string): Promise<{ outputUrl: string | null }> {
  const response = await fetch(`${API_BASE_URL}/api/blog/images/${requestId}`);
  if (!response.ok) {
    throw new Error('Failed to get image generation status');
  }
  return response.json();
}

export async function getImageGenerationStatus(requestId: string): Promise<{ outputUrl: string | null }> {
  const response = await fetch(`${API_BASE_URL}/api/blog/images/${requestId}/status`);
  if (!response.ok) {
    throw new Error('Failed to get image generation status');
  }
  return response.json();
}

// Article context operations
export async function updateArticleWithContext(articleId: number): Promise<{ content: string; success: boolean }> {
  const response = await fetch(`${API_BASE_URL}/api/blog/articles/${articleId}/context`, {
    method: 'PUT',
  });
  if (!response.ok) {
    throw new Error('Failed to update article with context');
  }
  return response.json();
} 
