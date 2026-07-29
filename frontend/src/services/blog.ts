import { ArticleListItem, ArticleData, RecommendedArticle, ArticleVersion, ArticleVersionListResponse } from '@/services/types';
import { Articles, Images } from '@/client';
import type { ArticleVersionListResponse as GeneratedArticleVersionListResponse } from '@/client';
import { generatedData } from '@/services/generatedClient';

type GetArticlesResponse = {
  articles: ArticleListItem[];
  total_pages: number;
  include_drafts: boolean;
};

// Article listing and search
export async function getArticles(
  page: number, 
  tag: string | null = null, 
  status: 'all' | 'published' | 'drafts' = 'published', 
  articlesPerPage?: number,
  sortBy?: string,
  sortOrder?: 'asc' | 'desc'
): Promise<GetArticlesResponse> {
  const data = await generatedData<GetArticlesResponse>(
    Articles.getArticles({
      query: {
        page,
        tag: tag && tag !== 'All' ? tag : undefined,
        status,
        articlesPerPage,
        sortBy,
        sortOrder,
      },
    }),
  );
  
  // Debug: Log API response
  console.log('articlesPayload API response:', {
    totalArticles: data.articles.length,
    drafts: data.articles.filter(a => !a.article.published_at).length,
    published: data.articles.filter(a => a.article.published_at).length,
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
  const data = await generatedData<GetArticlesResponse>(
    Articles.searchArticles({
      query: {
        query,
        page,
        tag: tag && tag !== 'All' ? tag : undefined,
        status,
        sortBy,
        sortOrder,
      },
    }),
  );
  
  return {
    articles: data.articles,
    total_pages: data.total_pages,
    include_drafts: data.include_drafts
  };
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  // Public endpoint - skip auth
  return generatedData<{ tags: string[] }>(Articles.getPopularTags());
}

// Article CRUD operations
export async function getArticle(slug: string): Promise<ArticleListItem | null> {
  try {
    // Public endpoint - skip auth
    return await generatedData<ArticleListItem>(
      Articles.getArticleData({ path: { slug } }),
    );
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
    return await generatedData<ArticleListItem>(
      Articles.getArticleData({ path: { slug: blogId } }),
    );
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
  publish: boolean;  // true = publish immediately, false = save as draft only
  authorId: string;
}): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  return generatedData<ArticleListItem>(Articles.createArticle({ body: article }));
}

export async function updateArticle(slug: string, article: {
  title: string;       // Updates draft_title
  content: string;     // Updates draft_content
  image_url?: string;  // Updates draft_image_url
  tags: string[];
}): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  // Updates always go to draft_* fields; use publishArticle() to publish
  return generatedData<ArticleListItem>(
    Articles.updateArticle({ path: { slug }, body: article }),
  );
}

// Article image operations
export async function generateArticleImage(prompt: string, articleId: string): Promise<{ success: boolean; generationRequestId: string }> {
  // Protected endpoint - requires auth
  const result = await generatedData<{ request_id: string }>(
    Images.generateImage({
      body: { prompt, article_id: articleId, generate_prompt: false },
    }),
  );
  return { 
    success: true, 
    generationRequestId: result.request_id
  };
}

export async function getImageGeneration(requestId: string): Promise<{ outputUrl: string | null }> {
  // Protected endpoint - requires auth
  const result = await generatedData<{ output_url?: string }>(
    Images.getImageGeneration({ path: { requestId } }),
  );
  return {
    outputUrl: result.output_url || null
  };
}

export async function getImageGenerationStatus(requestId: string): Promise<{ outputUrl: string | null }> {
  // Protected endpoint - requires auth
  const result = await generatedData<{ output_url?: string; outputUrl?: string }>(
    Images.getImageGenerationStatus({ path: { requestId } }),
  );
  return {
    outputUrl: result.output_url || result.outputUrl || null
  };
}

// Article context operations
export async function updateArticleWithContext(articleId: string): Promise<{ content: string; success: boolean }> {
  const article = await generatedData<{ draft_content: string }>(
    Articles.updateArticleWithContext({ path: { id: articleId } }),
  );
  return { content: article.draft_content, success: true };
}

export async function getArticleData(slug: string): Promise<ArticleData | null> {
  try {
    // Public endpoint - skip auth
    return await generatedData<ArticleData>(
      Articles.getArticleData({ path: { slug } }),
    );
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    throw error;
  }
}

export async function getRecommendedArticles(currentArticleId: string): Promise<RecommendedArticle[] | null> {
  // Public endpoint - skip auth
  return generatedData<RecommendedArticle[]>(
    Articles.getRecommendedArticles({ path: { id: currentArticleId } }),
  );
}

export async function deleteArticle(id: string): Promise<{ success: boolean }> {
  // Protected endpoint - requires auth
  return generatedData<{ success: boolean }>(
    Articles.deleteArticle({ path: { slug: id } }),
  );
}

// Version management operations

/**
 * Publish the current draft - copies draft_* fields to published_* fields.
 * Optionally pass a publishedAt date to override the default (now).
 */
export async function publishArticle(slug: string, publishedAt?: Date): Promise<ArticleListItem> {
  const body = publishedAt
    ? { published_at: Math.floor(publishedAt.getTime() / 1000) }
    : undefined;
  return generatedData<ArticleListItem>(
    Articles.publishArticle({ path: { slug }, body }),
  );
}

/**
 * Unpublish an article - clears published_* fields and published_at
 */
export async function unpublishArticle(slug: string): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  return generatedData<ArticleListItem>(
    Articles.unpublishArticle({ path: { slug } }),
  );
}

/**
 * List all versions for an article
 */
export async function listArticleVersions(slug: string): Promise<ArticleVersionListResponse> {
  // Protected endpoint - requires auth
  const data = await generatedData<GeneratedArticleVersionListResponse>(
    Articles.listArticleVersions({ path: { slug } }),
  );
  return {
    versions: data.versions as ArticleVersion[],
    draft_count: data.versions.filter((version) => version.status === 'draft').length,
    published_count: data.versions.filter((version) => version.status === 'published').length,
  };
}

/**
 * Get a specific version by ID
 */
export async function getArticleVersion(versionId: string): Promise<ArticleVersion> {
  // Protected endpoint - requires auth
  return generatedData<ArticleVersion>(
    Articles.getArticleVersion({ path: { versionId } }),
  );
}

/**
 * Revert to a previous version - creates a new draft from that version
 */
export async function revertToVersion(slug: string, versionId: string): Promise<ArticleListItem> {
  // Protected endpoint - requires auth
  return generatedData<ArticleListItem>(
    Articles.revertArticleToVersion({ path: { slug, versionId } }),
  );
}
