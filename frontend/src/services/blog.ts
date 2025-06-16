import { ArticleListItem, ArticleData, RecommendedArticle, ArticleRow  } from '@/services/types';
import { GetArticlesResponse } from '@/routes/dashboard/blog/index';
import { VITE_API_BASE_URL } from '@/services/constants';

// Article listing and search
export async function getArticles(page: number, tag: string | null = null): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const params = new URLSearchParams({
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {})
  });

  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles?${params}`);
  if (!response.ok) {
    throw new Error('Failed to fetch articles');
  }
  const data: GetArticlesResponse = await response.json();
  
  // Transform to ArticleListItem format
  return {
    articles: data.articles.map(item => ({
      article: {
        id: item.id,
        title: item.title || '',
        slug: item.slug || '',
        content: item.content,
        image: item.image,
        created_at: item.created_at,
        updated_at: item.created_at,
        published_at: item.published_at,
        is_draft: item.is_draft,
        image_generation_request_id: item.image_generation_request_id,
        author: null,
        chat_history: null
      },
      author: { id: 0, name: item.author },
      tags: item.tags.map(tag => ({ article_id: item.id, tag_id: 0, tag_name: tag }))
    })),
    totalPages: data.totalPages
  };
}

export async function searchArticles(query: string, page: number = 1, tag: string | null = null): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const params = new URLSearchParams({
    query,
    page: page.toString(),
    ...(tag && tag !== 'All' ? { tag } : {})
  });

  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/search?${params}`);
  if (!response.ok) {
    throw new Error('Failed to search articles');
  }
  const data: GetArticlesResponse = await response.json();
  
  // Transform to ArticleListItem format
  return {
    articles: data.articles.map(item => ({
      article: {
        id: item.id,
        title: item.title || '',
        slug: item.slug || '',
        content: item.content,
        image: item.image,
        created_at: item.created_at,
        updated_at: item.created_at,
        published_at: item.published_at,
        is_draft: item.is_draft,
        image_generation_request_id: item.image_generation_request_id,
        author: null,
        chat_history: null
      },
      author: { id: 0, name: item.author },
      tags: item.tags.map(tag => ({ article_id: item.id, tag_id: 0, tag_name: tag }))
    })),
    totalPages: data.totalPages
  };
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/tags/popular`);
  if (!response.ok) {
    throw new Error('Failed to fetch popular tags');
  }
  return response.json();
}

// Article CRUD operations
export async function getArticle(slug: string): Promise<ArticleListItem | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${slug}`);
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error('Failed to fetch article');
  }
  return response.json();
}

export async function getArticleById(blogId: string): Promise<ArticleListItem | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${blogId}`);
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
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles`, {
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
  if (!response.ok) {
    throw new Error('Failed to update article');
  }
  return response.json();
}

// Article image operations
export async function generateArticleImage(prompt: string, articleId: number): Promise<{ success: boolean; generationRequestId: string }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${articleId}/image`, {
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
  const response = await fetch(`${VITE_API_BASE_URL}/blog/images/${requestId}`);
  if (!response.ok) {
    throw new Error('Failed to get image generation status');
  }
  return response.json();
}

export async function getImageGenerationStatus(requestId: string): Promise<{ outputUrl: string | null }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/images/${requestId}/status`);
  if (!response.ok) {
    throw new Error('Failed to get image generation status');
  }
  return response.json();
}

// Article context operations
export async function updateArticleWithContext(articleId: number): Promise<{ content: string; success: boolean }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${articleId}/context`, {
    method: 'PUT',
  });
  if (!response.ok) {
    throw new Error('Failed to update article with context');
  }
  return response.json();
}

export async function getArticleData(slug: string): Promise<ArticleData | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${slug}`);
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error('Failed to fetch article data');
  }
  return response.json();
}

export async function getRecommendedArticles(currentArticleId: number): Promise<RecommendedArticle[] | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${currentArticleId}/recommended`);
  if (!response.ok) {
    throw new Error('Failed to fetch recommended articles');
  }
  return response.json();
}

export async function deleteArticle(id: number): Promise<{ success: boolean }> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/articles/${id}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete article');
  }
  return response.json();
}
