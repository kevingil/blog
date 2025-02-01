'use server'

import { eq, sql, or, like, and, inArray, desc } from 'drizzle-orm';
import { db } from '@/db/drizzle';
import { articles, users, articleTags, tags } from '@/db/schema';
import { ArticleListItem, ITEMS_PER_PAGE } from './index';
import { log } from 'console';


export async function getArticle(id: number): Promise<ArticleListItem> {
  const article = await db
    .select({
      id: articles.id,
      title: articles.title,
      slug: articles.slug,
      image: articles.image,
      content: articles.content,
      createdAt: articles.createdAt,
      publishedAt: articles.publishedAt,
      author: users.name,
    })
    .from(articles)
    .innerJoin(users, eq(articles.author, users.id))
    .where(eq(articles.id, id))
    .limit(1);

    const rawTagResults = await db
    .select({ name: tags.name })
    .from(articleTags)
    .where(eq(articleTags.articleId, id))
    .innerJoin(tags, eq(articleTags.tagId, tags.id));

  // Map tag results as strings
  const tagResults: { name: string }[] = rawTagResults.map(tag => ({
    name: tag.name || '' 
  }));

  return { 
    ...article[0], 
    tags: tagResults.map(tag => tag.name)
  };
}


export async function getArticles(page: number, tag: string | null = null): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const offset = (page - 1) * ITEMS_PER_PAGE;

  // Base conditions
  const conditions: any[] = [eq(articles.isDraft, false)];

  if (tag !== null && tag !== 'All') {
    
    const tagResult = await db
      .select({ id: tags.id })
      .from(tags)
      .where(eq(tags.name, tag));

    const tagId = tagResult[0]?.id;

    if (!tagId) {
      return { articles: [], totalPages: 0 };
    }
    console.log("tagId", tagId);

    // Filter articles by tag ID
    conditions.push(eq(articleTags.tagId, tagId));
  }

  // Main articles query
  const articlesData = await db
    .select({
      id: articles.id,
      title: articles.title,
      slug: articles.slug,
      image: articles.image,
      content: articles.content,
      createdAt: articles.createdAt,
      publishedAt: articles.publishedAt,
      author: users.name,
    })
    .from(articles)
    .innerJoin(users, eq(articles.author, users.id))
    .leftJoin(articleTags, eq(articles.id, articleTags.articleId))
    .where(and(...conditions))
    .groupBy(articles.id)
    .limit(ITEMS_PER_PAGE)
    .offset(offset)
    .orderBy(desc(articles.publishedAt));

  // For pagination
  const totalArticlesResult = await db
    .select({ count: sql<number>`count(DISTINCT ${articles.id})` })
    .from(articles)
    .leftJoin(articleTags, eq(articles.id, articleTags.articleId))
    .where(and(...conditions));

  const totalArticles = Number(totalArticlesResult[0].count);
  const totalPages = Math.ceil(totalArticles / ITEMS_PER_PAGE);

  // Add tags to articles
  const articlesWithTags = await Promise.all(
    articlesData.map(async (article) => {
      const tagData = await db
        .select({ tagName: tags.name })
        .from(articleTags)
        .innerJoin(tags, eq(articleTags.tagId, tags.id))
        .where(eq(articleTags.articleId, article.id));

      return {
        ...article,
        tags: tagData.map((tag) => tag.tagName),
      };
    })
  );

  return { articles: articlesWithTags, totalPages };
}


export async function searchArticles(
  query: string,
  page: number = 1,
  tag: string | null = null
): Promise<{ articles: ArticleListItem[], totalPages: number }> {
  const offset = (page - 1) * ITEMS_PER_PAGE;
  let searchResults: Set<number> = new Set();

  const textSearchResults = await db
    .select({
      id: articles.id,
    })
    .from(articles)
    .where(
      and(
        eq(articles.isDraft, false),
        or(
          like(articles.title, `%${query}%`),
          like(articles.content, `%${query}%`)
        )
      )
    )
    .limit(ITEMS_PER_PAGE);

  // Combine results
  textSearchResults.forEach(result => searchResults.add(result.id));

  // Get full article data for all found IDs
  const articlesData = await db
    .select({
      id: articles.id,
      title: articles.title,
      slug: articles.slug,
      image: articles.image,
      content: articles.content,
      createdAt: articles.createdAt,
      publishedAt: articles.publishedAt,
      author: users.name,
    })
    .from(articles)
    .leftJoin(users, eq(articles.author, users.id))
    .where(
      and(
        eq(articles.isDraft, false),
        inArray(articles.id, Array.from(searchResults))
      )
    )
    .orderBy(desc(articles.publishedAt))
    .limit(ITEMS_PER_PAGE)
    .offset(offset);

  // Get total count for pagination
  const totalArticles = await db
    .select({ count: sql<number>`count(*)` })
    .from(articles)
    .where(
      and(
        eq(articles.isDraft, false),
        or(
          like(articles.title, `%${query}%`),
          like(articles.content, `%${query}%`)
        )
      )
    );

  const totalPages = Math.ceil(Number(totalArticles[0].count) / ITEMS_PER_PAGE);

  // Add tags to articles
  const articlesWithTags = await Promise.all(
    articlesData.map(async (article) => {
      const tagData = await db
        .select({ tagName: tags.name })
        .from(articleTags)
        .innerJoin(tags, eq(articleTags.tagId, tags.id))
        .where(eq(articleTags.articleId, article.id));

      return {
        ...article,
        tags: tagData.map((tag) => tag.tagName),
      };
    })
  );

  return { articles: articlesWithTags, totalPages };
}

export async function getPopularTags(): Promise<{ tags: string[] }> {
  const popularTags = await db
    .select({
      tagName: tags.name,
    })
    .from(articleTags)
    .innerJoin(tags, eq(articleTags.tagId, tags.id))
    .innerJoin(articles, eq(articleTags.articleId, articles.id))
    .where(eq(articles.isDraft, false))
    .groupBy(tags.name)
    .orderBy(desc(sql<number>`count(*)`))
    .limit(10);

  return { tags: popularTags.map((tag) => tag.tagName || '') };
}
