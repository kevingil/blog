import { Article, ArticleChatHistory } from '../types';
import { VITE_API_BASE_URL } from '../constants';
import { handleApiResponse, getAuthHeadersWithContentType } from '../apiHelpers';

export interface GenerateSessionResponse {
  article: Article;
  request_id: string;
}

export async function generateArticle(prompt?: string, title?: string): Promise<GenerateSessionResponse> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/generate`, {
    method: 'POST',
    headers: getAuthHeadersWithContentType(),
    body: JSON.stringify({ prompt: prompt || '', title: title || '' }),
  });

  return handleApiResponse<GenerateSessionResponse>(response);
}

export async function getArticleChatHistory(articleId: number): Promise<ArticleChatHistory | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/${articleId}/chat-history`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 404) {
      return null;
    }
    throw new Error('Failed to get article chat history');
  }

  return handleApiResponse<ArticleChatHistory>(response);
}

export async function updateWithContext(articleId: number): Promise<{ content: string, success: boolean } | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/blog/${articleId}/update`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 404) {
      return null;
    }
    throw new Error('Failed to update article with context');
  }

  const article = await handleApiResponse<Article>(response);
  return {
    content: article.content,
    success: true,
  };
}
