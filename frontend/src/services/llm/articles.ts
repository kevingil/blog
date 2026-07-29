import { Article } from '../types';
import { Articles } from '@/client';
import { generatedData } from '../generatedClient';

export interface GenerateSessionResponse {
  article: Article;
  request_id: string;
}

export async function generateArticle(prompt?: string, title?: string): Promise<GenerateSessionResponse> {
  return generatedData<GenerateSessionResponse>(
    Articles.generateArticle({
      body: { prompt: prompt || '', title: title || '' },
    }),
  );
}

export async function updateWithContext(articleId: number): Promise<{ content: string, success: boolean } | null> {
  try {
    const article = await generatedData<Article>(
      Articles.updateArticleWithContext({ path: { id: String(articleId) } }),
    );
    return { content: article.draft_content, success: true };
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    throw error;
  }
}
