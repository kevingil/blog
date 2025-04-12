'use server'

import { eq } from 'drizzle-orm';
import { db } from '@/db/drizzle';
import { Article, articles, articleTags, imageGeneration, ImageGeneration, NewArticle, tags } from '@/db/schema';
import { v4 as uuidv4 } from 'uuid';


export async function createArticle({
    title,
    content,
    image,
    tags: newTags,
    isDraft,
    authorId,
  }: {
    title: string;
    content: string;
    image?: string;
    tags: string[];
    isDraft: boolean;
    authorId: number;
  }) {

    let slug = title
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, ''); 

    const existingArticle = await db.select().from(articles).where(eq(articles.slug, slug)).limit(1);

    if (existingArticle.length > 0) {
      slug = slug + '-' + uuidv4().substring(0, 4);
    }

    const newArticle: NewArticle = {
      image: image || null,
      slug: slug,
      title: title,
      content: content,
      isDraft: isDraft,
      embedding: null,
      author: authorId,
      createdAt: new Date().getTime(),
      updatedAt: new Date().getTime(),
    }

    try {
      const postedArticle = await db.insert(articles).values(newArticle).returning();
  
      // Add tags
      for (const tagName of newTags) {
        let tag = await db.select().from(tags).where(eq(tags.name, tagName)).limit(1);
        if (tag.length === 0) {
          const [newTag] = await db.insert(tags).values({ name: tagName }).returning();
          tag = [newTag];
        }
        await db.insert(articleTags).values({ articleId: postedArticle[0].id, tagId: tag[0].id });
      }

      return postedArticle[0] as Article;


    } catch (error) {
      console.error('Error creating article', error);
      throw new Error('Failed to create article');
    }
    
}


export async function getArticle(slug: string) {
  const article = await db.select().from(articles).where(eq(articles.slug, slug)).limit(1);
  
  if (article.length === 0) {
    return null;
  }

  const articleTagsData = await db
    .select({ name: tags.name })
    .from(tags)
    .innerJoin(articleTags, eq(articleTags.tagId, tags.id))
    .where(eq(articleTags.articleId, article[0].id));

  return {
    ...article[0],
    tags: articleTagsData,
  };
}

export async function updateArticle({
  slug,
  title,
  content,
  image,
  tags: newTags,
  isDraft,
  publishedAt,
}: {
  slug: string;
  title: string;
  content: string;
  image?: string;
  tags: string[];
  isDraft: boolean;
  publishedAt: number;
}) {

  // Get the article
  const article = await db.select().from(articles).where(eq(articles.slug, slug)).limit(1);
  if (article.length === 0) {
    throw new Error('Article not found');
  }
  const articleId = article[0].id;

  // Update published if changed from draft to published
  // and if the publishedAt wasn't set, then set it to the current time
  const publishedAtCheck = 
  isDraft != true
  ?((article[0].isDraft == true) && (isDraft == false) && (article[0].publishedAt == publishedAt))
  ? new Date().getTime() 
  : publishedAt
  : null;


  // If slug is changed, check if it already exists, if so, add random number to make it unique
  const existingSlug = article[0].slug;
  let newSlug = title
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  if (newSlug !== existingSlug) {
    const existingArticle = await db.select().from(articles).where(eq(articles.slug, newSlug)).limit(1);
    if (existingArticle.length > 0) {
      newSlug = newSlug + '-' + uuidv4().substring(0, 4);
    }
  }

  // Update article
  await db.update(articles).set({ 
    slug: newSlug,
    title: title, 
    content: content, 
    image: image, 
    isDraft: isDraft,
    updatedAt: new Date().getTime(),
    publishedAt: publishedAtCheck,
  }).where(eq(articles.id, articleId));

  // Delete existing tags
  await db.delete(articleTags).where(eq(articleTags.articleId, articleId));

  // Add new tags
  for (const tagName of newTags) {
    let tag = await db.select().from(tags).where(eq(tags.name, tagName)).limit(1);
    
    if (tag.length === 0) {
      const [newTag] = await db.insert(tags).values({ name: tagName }).returning();
      tag = [newTag];
    }

    await db.insert(articleTags).values({ articleId, tagId: tag[0].id });
  }
}
