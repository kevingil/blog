export interface Article {
  id: number;
  image_url: string;
  slug: string;
  title: string;
  content: string;
  author: number;
  created_at: string;
  updated_at: string;
  is_draft: boolean;
  image_generation_request_id: string;
  published_at?: string | null;
  chat_history: string;
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
  article_id: number;
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
    title: string;
    slug: string;
    content: string;
    image_url: string;
    created_at: string;
    updated_at: string;
    published_at: string | null;
    is_draft: boolean;
    image_generation_request_id?: string | null;
    author_id: string | null;
    chat_history?: any | null;
    tag_ids?: number[];
    imagen_request_id?: string | null;
    embedding?: any | null;
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
    id: number;
    name: string;
  };
}

export type TagData = {
  article_id: number;
  tag_id: number;
  tag_name: string | null;
}

export type RecommendedArticle = {
  id: number;
  title: string;
  slug: string;
  image_url: string | null;
  published_at: string | null;
  created_at: string;
  author: string | null;
}

export type ArticleRow = {
  id: number;
  title: string | null;
  content: string | null;
  created_at: string;
  published_at: string | null;
  is_draft: boolean;
  slug: string | null;
  tags: string[];
  image_url: string | null;
} 
