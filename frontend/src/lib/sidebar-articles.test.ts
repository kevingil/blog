import { describe, expect, test } from 'bun:test';
import {
  listItemFromArticle,
  patchSidebarArticleTitle,
  upsertSidebarArticle,
  type SidebarArticlesPage,
} from './sidebar-articles';
import type { InfiniteData } from '@tanstack/react-query';

function item(id: string, title: string) {
  return listItemFromArticle(
    {
      id,
      slug: title.toLowerCase().replace(/\s+/g, '-'),
      draft_title: title,
      created_at: '2026-10-06T00:00:00.000Z',
    },
    'Kevin Gil',
  );
}

describe('sidebar article cache', () => {
  test('inserts a new article at the top of the first page', () => {
    const existing = item('old', 'Older draft');
    const created = item('new', 'Untitled Article');
    const data: InfiniteData<SidebarArticlesPage> = {
      pages: [{ articles: [existing], total_pages: 1 }],
      pageParams: [1],
    };
    const next = upsertSidebarArticle(data, created);
    expect(next.pages[0].articles.map((entry) => entry.article.id)).toEqual(['new', 'old']);
  });

  test('replaces an article that is already in the list', () => {
    const created = item('new', 'Untitled Article');
    const renamed = item('new', 'RSI is a systems problem');
    const next = upsertSidebarArticle(
      { pages: [{ articles: [created], total_pages: 1 }], pageParams: [1] },
      renamed,
    );
    expect(next.pages[0].articles).toHaveLength(1);
    expect(next.pages[0].articles[0].article.draft_title).toBe('RSI is a systems problem');
  });

  test('patches the title without refetching the list', () => {
    const created = item('new', 'Untitled Article');
    const next = patchSidebarArticleTitle(
      { pages: [{ articles: [created], total_pages: 1 }], pageParams: [1] },
      'new',
      'RSI is a systems problem',
    );
    expect(next?.pages[0].articles[0].article.draft_title).toBe('RSI is a systems problem');
    expect(next?.pages[0].articles[0].article.slug).toBe(created.article.slug);
  });
});
