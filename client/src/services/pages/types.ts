
export type ContentType = 'html' | 'markdown' | 'plain_text';

export interface PageContent {
  id: number;
  page_id: number;
  content: string;
  content_type: ContentType;
  created_at: string;
  updated_at: string | null;
}

export interface Page {
  id: number;
  slug: string;
  title: string;
  meta_description: string | null;
  is_active: boolean;
  is_system_page: boolean;
  show_in_nav: boolean;
  nav_order: number | null;
  created_at: string;
  updated_at: string;
  current_content?: PageContent;
}

// API Response Types
export interface ApiResponse {
  success: boolean;
  message?: string;
}

export interface PageResponse extends ApiResponse {
  page: Page;
}

export interface PagesResponse extends ApiResponse {
  pages: Page[];
  total: number;
}

// Request Types
export interface CreatePageData {
  slug: string;
  title: string;
  meta_description?: string | null;
  content: string;
  content_type: ContentType;
  is_active?: boolean;
  show_in_nav?: boolean;
  nav_order?: number | null;
}

export interface UpdatePageData extends Partial<Omit<CreatePageData, 'slug'>> {
  id: number;
}

export interface UpdatePageOrderData {
  id: number;
  nav_order: number;
}

export interface UpdatePageStatusData {
  id: number;
  is_active: boolean;
}
