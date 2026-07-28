export interface Article {
  id: string;
  slug: string;
  author_id: string;
  
  // Draft content (always present)
  draft_title: string;
  draft_content: string;
  draft_image_url: string;
  
  // Published content (null if unpublished)
  published_title: string | null;
  published_content: string | null;
  published_image_url: string | null;
  published_at: string | null;
  
  // Version pointers
  current_draft_version_id: string | null;
  current_published_version_id: string | null;
  
  // Metadata
  image_generation_request_id?: string | null;
  session_memory?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

// Version history types
export interface ArticleVersion {
  id: string;
  article_id: string;
  version_number: number;
  status: 'draft' | 'published';
  title: string;
  content: string;
  image_url: string;
  edited_by: string;
  created_at: string;
}

export interface ArticleVersionListResponse {
  versions: ArticleVersion[];
  draft_count: number;
  published_count: number;
}

// Helper functions for article display
export function isPublished(article: Article | ArticleListItem['article'] | null | undefined): boolean {
  return article?.published_at !== null && article?.published_at !== undefined;
}

export function getDisplayTitle(article: Article | ArticleListItem['article'], preferPublished = true): string {
  if (preferPublished && article.published_title) return article.published_title;
  return article.draft_title;
}

export function getDisplayContent(article: Article | ArticleListItem['article'], preferPublished = true): string {
  if (preferPublished && article.published_content) return article.published_content;
  return article.draft_content;
}

export function getDisplayImageUrl(article: Article | ArticleListItem['article'], preferPublished = true): string {
  if (preferPublished && article.published_image_url) return article.published_image_url;
  return article.draft_image_url;
}

export function hasDraftChanges(article: Article | ArticleListItem['article'] | null | undefined): boolean {
  if (!article || !article.published_at) return false;
  return article.draft_title !== article.published_title || 
         article.draft_content !== article.published_content ||
         article.draft_image_url !== article.published_image_url;
}

export interface ArticleChatHistoryMessage {
  role: "user" | "assistant" | "system";
  content: string;
  created_at: number;
  metadata: Record<string, unknown>;
}

export interface ArticleChatHistory {
  messages: ArticleChatHistoryMessage[];
  metadata: Record<string, unknown>;
}

export interface Tag {
  id: number;
  name: string;
}

export interface ArticleTag {
  article_id: string;
  tag_id: number;
}

export interface ImageGeneration {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at?: number;
  prompt: string;
  provider: string;
  model: string;
  request_id: string;
  output_url?: string;
  storage_key?: string;
}

export interface AboutPage {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at?: number;
  title: string;
  content: string;
  profile_image?: string;
  meta_description?: string;
  last_updated: string;
}

export interface ContactPage {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at?: number;
  title: string;
  content: string;
  email_address: string;
  social_links?: string;
  meta_description?: string;
  last_updated: string;
}

export interface Project {
  id: number;
  created_at: number;
  updated_at: number;
  deleted_at?: number;
  title: string;
  description: string;
  url?: string;
  image?: string;
}

export interface User {
  id: number;
  name: string;
  email: string;
  role: string;
}

export interface ImageGenerationStatus {
  accepted: boolean;
  requestId: string;
  outputUrl: string;
} 

export const ITEMS_PER_PAGE = 6;

export type ArticleListItem = {
  article: {
    id: string;
    slug: string;
    author_id: string | null;
    
    // Draft content (always present)
    draft_title: string;
    draft_content: string;
    draft_image_url: string;
    
    // Published content (null if unpublished)
    published_title: string | null;
    published_content: string | null;
    published_image_url: string | null;
    published_at: string | null;
    
    // Version pointers
    current_draft_version_id: string | null;
    current_published_version_id: string | null;
    
    // Metadata
    created_at: string;
    updated_at: string;
    image_generation_request_id?: string | null;
    tag_ids?: number[];
    imagen_request_id?: string | null;
    session_memory?: Record<string, any>;
  };
  author: {
    id: string;
    name: string;
  };
  tags: {
    article_id: string;
    tag_id: number;
    name: string;
  }[] | null;
};


export type ArticleData = {
  article: Article;
  tags: TagData[] | null;
  author: {
    id: string;
    name: string;
  };
}

export type TagData = {
  article_id: string;
  tag_id: number;
  tag_name: string | null;
}

export type RecommendedArticle = {
  id: string;
  title: string; // Backend returns the appropriate title (published if available)
  slug: string;
  image_url: string | null;
  published_at: string | null;
  created_at: string;
  author: string | null;
}

export type ArticleRow = {
  id: string;
  slug: string | null;
  draft_title: string | null;
  draft_content: string | null;
  draft_image_url: string | null;
  published_title: string | null;
  published_content: string | null;
  published_image_url: string | null;
  published_at: string | null;
  created_at: string;
  tags: string[];
}

export interface ArticleSource {
  id: string;
  article_id: string;
  title: string;
  content: string;
  url: string;
  source_type: 'web' | 'manual' | 'pdf';
  embedding?: number[];
  meta_data?: Record<string, any>;
  created_at: string;
}

export interface ArticleSourceWithArticle extends ArticleSource {
  content_preview: string;
  article_title: string;
  article_slug: string;
}

export interface CreateSourceRequest {
  article_id: string;
  title: string;
  content: string;
  url?: string;
  source_type?: 'web' | 'manual';
}

export interface UpdateSourceRequest {
  title?: string;
  content?: string;
  url?: string;
  source_type?: 'web' | 'manual' | 'pdf';
}

export interface ScrapeSourceRequest {
  article_id: string;
  url: string;
} 
