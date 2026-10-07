import type { InfiniteData } from '@tanstack/react-query';
import type { ArticleListItem } from '@/services/types';

export type SidebarArticlesPage = {
  articles: ArticleListItem[];
  total_pages: number;
  include_drafts?: boolean;
};

type ArticleSeed = {
  id: string;
  slug: string;
  author_id?: string | null;
  draft_title?: string;
  draft_content?: string;
  draft_image_url?: string;
  created_at?: string;
  updated_at?: string;
  published_at?: string | null;
  external_url?: string | null;
};

export function listItemFromArticle(article: ArticleSeed, authorName: string): ArticleListItem {
  const createdAt = article.created_at || new Date().toISOString();
  return {
    article: {
      id: article.id,
      slug: article.slug,
      author_id: article.author_id ?? null,
      draft_title: article.draft_title || 'Untitled Article',
      draft_content: article.draft_content || '',
      draft_image_url: article.draft_image_url || '',
      published_title: null,
      published_content: null,
      published_image_url: null,
      published_at: article.published_at ?? null,
      current_draft_version_id: null,
      current_published_version_id: null,
      created_at: createdAt,
      updated_at: article.updated_at || createdAt,
      external_url: article.external_url ?? null,
    },
    author: {
      id: article.author_id || '',
      name: authorName,
    },
    tags: [],
  };
}

export function upsertSidebarArticle(
  data: InfiniteData<SidebarArticlesPage> | undefined,
  item: ArticleListItem,
): InfiniteData<SidebarArticlesPage> {
  if (!data?.pages?.length) {
    return {
      pages: [{ articles: [item], total_pages: 1, include_drafts: true }],
      pageParams: [1],
    };
  }
  const pages = data.pages.map((page, index) => {
    const articles = page.articles.filter((entry) => entry.article.id !== item.article.id);
    if (index === 0) {
      return { ...page, articles: [item, ...articles] };
    }
    return { ...page, articles };
  });
  return { ...data, pages };
}

export function patchSidebarArticleTitle(
  data: InfiniteData<SidebarArticlesPage> | undefined,
  articleId: string,
  title: string,
): InfiniteData<SidebarArticlesPage> | undefined {
  if (!data?.pages) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      articles: page.articles.map((entry) =>
        entry.article.id === articleId
          ? { ...entry, article: { ...entry.article, draft_title: title } }
          : entry,
      ),
    })),
  };
}
