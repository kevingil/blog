'use server'
import { Article, ArticleChatHistory } from '../types';
import { API_BASE_URL } from '../constants';
export async function generateArticle(prompt: string, title: string, authorId: number, draft?: boolean) {
  const response = await fetch(`${API_BASE_URL}/blog/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      user_id: authorId,
      prompt,
      title,
      is_draft: draft || true,
    }),
  });

  if (!response.ok) {
    throw new Error('Failed to generate article');
  }

  const article: Article = await response.json();
  return article;
}

export async function getArticleChatHistory(articleId: number): Promise<ArticleChatHistory | null> {
  const response = await fetch(`${API_BASE_URL}/blog/${articleId}/chat-history`, {
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

  const history: ArticleChatHistory = await response.json();
  return history;
}

export async function updateWithContext(articleId: number): Promise<{ content: string, success: boolean } | null> {
  const response = await fetch(`${API_BASE_URL}/blog/${articleId}/update`, {
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

  const article: Article = await response.json();
  return {
    content: article.content,
    success: true,
  };
}
