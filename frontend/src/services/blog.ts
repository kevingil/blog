import { ArticleListItem, ArticleData, RecommendedArticle  } from '@/services/types';
import { GetArticlesResponse } from '@/routes/dashboard/blog/index';
import { VITE_API_BASE_URL } from '@/services/constants';
import { apiGet, apiPost, apiPut, apiDelete, authenticatedFetch } from '@/services/authenticatedFetch';

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

  // Public endpoint - skip auth
  const data = await apiGet<GetArticlesResponse>(`/blog/articles?${params}`, { skipAuth: true });
  
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

  // Public endpoint - skip auth
  const data = await apiGet<GetArticlesResponse>(`/blog/articles/search?${params}`, { skipAuth: true });
  
  return {
    articles: data.articles,
    total_pages: data.total_pages,
    include_drafts: data.include_drafts
  };
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  // Public endpoint - skip auth
  return apiGet<{ tags: string[] }>('/blog/tags/popular', { skipAuth: true });
}

// Article CRUD operations
export async function getArticle(slug: string): Promise<ArticleListItem | null> {
  try {
    // Public endpoint - skip auth
    return await apiGet<ArticleListItem>(`/blog/articles/${slug}`, { skipAuth: true });
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    throw error;
  }
}

export async function getArticleById(blogId: string): Promise<ArticleListItem | null> {
  try {
    // Public endpoint - skip auth
    return await apiGet<ArticleListItem>(`/blog/articles/${blogId}`, { skipAuth: true });
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    throw error;
  }
}

export async function createArticle(article: {
  title: string;
  content: string;
  image_url?: string;
  tags: string[];
  isDraft: boolean;
  authorId: number;
}): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  return apiPost<ArticleListItem>('/blog/articles', article);
}

export async function updateArticle(slug: string, article: {
  title: string;
  content: string;
  image_url?: string;
  tags: string[];
  is_draft: boolean;
  published_at: number | null;
}): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  return apiPost<ArticleListItem>(`/blog/articles/${slug}/update`, article);
}

// Article image operations
export async function generateArticleImage(prompt: string, articleId: string): Promise<{ success: boolean; generationRequestId: string }> {
  // Protected endpoint - requires auth
  const result = await apiPost<{ request_id: string }>('/images/generate', {
    prompt,
    article_id: articleId,
    generate_prompt: false
  });
  return { 
    success: true, 
    generationRequestId: result.request_id
  };
}

export async function getImageGeneration(requestId: string): Promise<{ outputUrl: string | null }> {
  // Protected endpoint - requires auth
  const result = await apiGet<{ output_url?: string }>(`/images/${requestId}`);
  return {
    outputUrl: result.output_url || null
  };
}

export async function getImageGenerationStatus(requestId: string): Promise<{ outputUrl: string | null }> {
  // Protected endpoint - requires auth
  const result = await apiGet<{ output_url?: string; outputUrl?: string }>(`/images/${requestId}/status`);
  return {
    outputUrl: result.output_url || result.outputUrl || null
  };
}

// Article context operations
export async function updateArticleWithContext(articleId: string): Promise<{ content: string; success: boolean }> {
  // Protected endpoint - requires auth
  return apiPut<{ content: string; success: boolean }>(`/blog/articles/${articleId}/context`);
}

export async function getArticleData(slug: string): Promise<ArticleData | null> {
  try {
    // Public endpoint - skip auth
    return await apiGet<ArticleData>(`/blog/articles/${slug}`, { skipAuth: true });
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    throw error;
  }
}

export async function getRecommendedArticles(currentArticleId: number): Promise<RecommendedArticle[] | null> {
  // Public endpoint - skip auth
  return apiGet<RecommendedArticle[]>(`/blog/articles/${currentArticleId}/recommended`, { skipAuth: true });
}

export async function deleteArticle(id: string): Promise<{ success: boolean }> {
  // Protected endpoint - requires auth
  return apiDelete<{ success: boolean }>(`/blog/articles/${id}`);
}
