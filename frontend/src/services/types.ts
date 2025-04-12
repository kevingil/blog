export interface Article {
  id: number;
  image: string;
  slug: string;
  title: string;
  content: string;
  author: number;
  created_at: number;
  updated_at: number;
  is_draft: boolean;
  image_generation_request_id: string;
  published_at?: number;
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
  created_at: number;
  updated_at: number;
  deleted_at?: number;
  name: string;
  email: string;
  role: string;
}

export interface ImageGenerationStatus {
  accepted: boolean;
  requestId: string;
  outputUrl: string;
} 
