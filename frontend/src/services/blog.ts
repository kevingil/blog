import { ArticleListItem, ArticleData, RecommendedArticle  } from '@/services/types';
import { GetArticlesResponse } from '@/routes/dashboard/blog/index';
import { VITE_API_BASE_URL } from '@/services/constants';
import { handleApiResponse } from '@/services/apiHelpers';

// Article listing and search
export async function getArticles(
  page: number, 
  tag: string | null = null, 
  status: 'all' | 'published' | 'drafts' = 'published', 
  articlesPerPage?: number,
  sortBy?: string,
  sortOrder?: 'asc' | 'desc'
): Promise<GetArticlesResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {}),
    status,
    ...(articlesPerPage !== undefined ? { articlesPerPage: articlesPerPage.toString() } : {}),
    ...(sortBy ? { sortBy } : {}),
    ...(sortOrder ? { sortOrder } : {})
  });

  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles?${params}`);
  const data: GetArticlesResponse = await handleApiResponse<GetArticlesResponse>(response);
  
  // Debug: Log API response
  console.log('articlesPayload API response:', {
    totalArticles: data.articles.length,
    drafts: data.articles.filter(a => a.article.is_draft).length,
    published: data.articles.filter(a => !a.article.is_draft).length,
    status
  });
  
  return {
    articles: data.articles,
    total_pages: data.total_pages,
    include_drafts: data.include_drafts
  };
}

export async function searchArticles(
  query: string, 
  page: number = 1, 
  tag: string | null = null,
  status: 'all' | 'published' | 'drafts' = 'published',
  sortBy?: string,
  sortOrder?: 'asc' | 'desc'
): Promise<GetArticlesResponse> {
  const params = new URLSearchParams({
    query,
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {}),
    status,
    ...(sortBy ? { sortBy } : {}),
    ...(sortOrder ? { sortOrder } : {})
  });

  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/search?${params}`);
  const data: GetArticlesResponse = await handleApiResponse<GetArticlesResponse>(response);
  
  return {
    articles: data.articles,
    total_pages: data.total_pages,
    include_drafts: data.include_drafts
  };
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/tags/popular`);
  return handleApiResponse<{ tags: string[] }>(response);
}

// Article CRUD operations
export async function getArticle(slug: string): Promise<ArticleListItem | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${slug}`);
  if (response.status === 404) {
    return null;
  }
  return handleApiResponse<ArticleListItem>(response);
}

export async function getArticleById(blogId: string): Promise<ArticleListItem | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${blogId}`);
  if (response.status === 404) {
    return null;
  }
  return handleApiResponse<ArticleListItem>(response);
}

export async function createArticle(article: {
  title: string;
  content: string;
  image_url?: string;
  tags: string[];
  isDraft: boolean;
  authorId: number;
}): Promise<ArticleListItem> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(article),
  });
  return handleApiResponse<ArticleListItem>(response);
}

export async function updateArticle(slug: string, article: {
  title: string;
  content: string;
  image_url?: string;
  tags: string[];
  is_draft: boolean;
  published_at: number | null;
}): Promise<ArticleListItem> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${slug}/update`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(article),
  });
  return handleApiResponse<ArticleListItem>(response);
}

// Article image operations
export async function generateArticleImage(prompt: string, articleId: string): Promise<{ success: boolean; generationRequestId: string }> {
  const response = await fetch(`${VITE_API_BASE_URL}/images/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ 
      prompt,
      article_id: articleId,
      generate_prompt: false
    }),
  });
  const result = await handleApiResponse<{ request_id: string }>(response);
  return { 
    success: true, 
    generationRequestId: result.request_id
  };
}

export async function getImageGeneration(requestId: string): Promise<{ outputUrl: string | null }> {
  const response = await fetch(`${VITE_API_BASE_URL}/images/${requestId}`);
  const result = await handleApiResponse<{ output_url?: string }>(response);
  return {
    outputUrl: result.output_url || null
  };
}

export async function getImageGenerationStatus(requestId: string): Promise<{ outputUrl: string | null }> {
  const response = await fetch(`${VITE_API_BASE_URL}/images/${requestId}/status`);
  const result = await handleApiResponse<{ output_url?: string; outputUrl?: string }>(response);
  return {
    outputUrl: result.output_url || result.outputUrl || null
  };
}

// Article context operations
export async function updateArticleWithContext(articleId: string): Promise<{ content: string; success: boolean }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${articleId}/context`, {
    method: 'PUT',
  });
  return handleApiResponse<{ content: string; success: boolean }>(response);
}

export async function getArticleData(slug: string): Promise<ArticleData | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${slug}`);
  if (response.status === 404) {
    return null;
  }
  return handleApiResponse<ArticleData>(response);
}

export async function getRecommendedArticles(currentArticleId: number): Promise<RecommendedArticle[] | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${currentArticleId}/recommended`);
  return handleApiResponse<RecommendedArticle[]>(response);
}

export async function deleteArticle(id: string): Promise<{ success: boolean }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${id}`, {
    method: 'DELETE',
  });
  return handleApiResponse<{ success: boolean }>(response);
}
