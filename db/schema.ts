import { sql } from 'drizzle-orm';
import { integer, sqliteTable, text, unique, customType } from 'drizzle-orm/sqlite-core';

// Custom vector type for embeddings
export const float32Array = customType<{
  data: number[];
  config: { dimensions: number };
  configRequired: true;
  driverData: Buffer;
}>({
  dataType(config) {
    return `F32_BLOB(${config.dimensions})`;
  },
  fromDriver(value: Buffer) {
    return Array.from(new Float32Array(value.buffer));
  },
  toDriver(value: number[]) {
    return sql`vector32(${JSON.stringify(value)})`;
  },
});


// Role enum
export const Role = {
  USER: 'user',
  ADMIN: 'admin'
} as const;

export type Role = typeof Role[keyof typeof Role];


// Users table
export const users = sqliteTable('users', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  name: text('name').notNull(),
  email: text('email').notNull().unique(),
  passwordHash: text('passwordHash').notNull(),
  role: text('role').notNull().default(Role.USER),
  memory: text('memory'),
});

// Projects table
export const projects = sqliteTable('projects', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  title: text('title').notNull(),
  description: text('description').notNull(),
  url: text('url').notNull(),
  image: text('image'),
});

// About Page table
export const aboutPage = sqliteTable('about_page', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  title: text('title'),
  content: text('content'), 
  profileImage: text('profile_image'),
  metaDescription: text('meta_description'), 
  lastUpdated: text('last_updated'),
});

// Contact Page table
export const contactPage = sqliteTable('contact_page', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  title: text('title'),
  content: text('content'), 
  emailAddress: text('email_address'), 
  socialLinks: text('social_links'), 
  metaDescription: text('meta_description'), 
  lastUpdated: text('last_updated'),
});

// Articles table with vector embeddings
export const articles = sqliteTable('articles', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  image: text('image'),
  slug: text('slug').notNull().unique(),
  title: text('title').notNull(),
  content: text('content').notNull(),
  author: integer('author').notNull().references(() => users.id),
  createdAt: integer('created_at').notNull().default(sql`(unixepoch())`),
  updatedAt: integer('updated_at').notNull().default(sql`(unixepoch())`),
  publishedAt: integer('published_at'),
  isDraft: integer('is_draft', { mode: 'boolean' }).notNull().default(false),
  embedding: float32Array('embedding', { dimensions: 1536 }),
  imageGenerationRequestId: text('image_generation_request_id'),
  chatHistory: text('chat_history'),
});

// Tags table
export const tags = sqliteTable('tags', {
  id: integer('tag_id').primaryKey({ autoIncrement: true }),
  name: text('tag_name').unique(),
});

// Article Tags junction table
export const articleTags = sqliteTable('article_tags', {
  articleId: integer('article_id').notNull().references(() => articles.id),
  tagId: integer('tag_id').notNull().references(() => tags.id),
}, (table) => ({
  pk: unique().on(table.articleId, table.tagId),
}));


// Image generation table
export const imageGeneration = sqliteTable('image_generation', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  prompt: text('prompt'),
  provider: text('provider'),
  model: text('model'),
  requestId: text('request_id'),
  outputUrl: text('output_url'),
  storageKey: text('storage_key'),
  createdAt: integer('created_at').notNull().default(sql`(unixepoch())`),
});


// Function to create vector index (to be used after table creation)
export async function createVectorIndex(db: any) {
  await db.run(sql`
    CREATE INDEX IF NOT EXISTS articles_embedding_idx
    ON articles(embedding)
    USING vector_cosine(1536)
  `);
}

// Helper function to insert vector data
export function insertVectorData(vector: number[]) {
  return sql`vector32(${JSON.stringify(vector)})`;
}

// Helper function for vector similarity search
export function vectorSimilaritySearch(vector: number[], limit: number) {
  return sql`
    SELECT id, title, content, vector_distance_cos(embedding, vector32(${JSON.stringify(vector)})) as distance
    FROM articles
    ORDER BY distance ASC
    LIMIT ${limit}
  `;
}


// Infer types
export type User = typeof users.$inferSelect;
export type NewUser = typeof users.$inferInsert;

export type Project = typeof projects.$inferSelect;
export type NewProject = typeof projects.$inferInsert;

export type AboutPage = typeof aboutPage.$inferSelect;
export type NewAboutPage = typeof aboutPage.$inferInsert;

export type ContactPage = typeof contactPage.$inferSelect;
export type NewContactPage = typeof contactPage.$inferInsert;

export type Article = typeof articles.$inferSelect;
export type NewArticle = typeof articles.$inferInsert;

export type Tag = typeof tags.$inferSelect;
export type NewTag = typeof tags.$inferInsert;

export type ArticleTag = typeof articleTags.$inferSelect;
export type NewArticleTag = typeof articleTags.$inferInsert;

export type ImageGeneration = typeof imageGeneration.$inferSelect;
export type NewImageGeneration = typeof imageGeneration.$inferInsert;
