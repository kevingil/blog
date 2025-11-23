import { apiGet, apiPost, apiPut, apiDelete } from '@/services/authenticatedFetch';

export interface Page {
  id: string;
  slug: string;
  title: string;
  content: string;
  description: string;
  image_url: string;
  meta_data: Record<string, any>;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface PageListResponse {
  pages: Page[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface PageCreateRequest {
  slug: string;
  title: string;
  content: string;
  description?: string;
  image_url?: string;
  meta_data?: Record<string, any>;
  is_published: boolean;
}

export interface PageUpdateRequest {
  title?: string;
  content?: string;
  description?: string;
  image_url?: string;
  meta_data?: Record<string, any>;
  is_published?: boolean;
}

// Dashboard CRUD operations (authenticated)
export async function getAllPages(page: number = 1, perPage: number = 20, isPublished?: boolean): Promise<PageListResponse> {
  const params = new URLSearchParams({
    page: page.toString(),
    perPage: perPage.toString(),
  });

  if (isPublished !== undefined) {
    params.append('isPublished', isPublished.toString());
  }

  return apiGet<PageListResponse>(`/dashboard/pages?${params}`);
}

export async function getPage(id: string): Promise<Page> {
  return apiGet<Page>(`/dashboard/pages/${id}`);
}

export async function createPage(data: PageCreateRequest): Promise<Page> {
  return apiPost<Page>('/dashboard/pages', data);
}

export async function updatePage(id: string, data: PageUpdateRequest): Promise<Page> {
  return apiPut<Page>(`/dashboard/pages/${id}`, data);
}

export async function deletePage(id: string): Promise<{ success: boolean }> {
  return apiDelete<{ success: boolean }>(`/dashboard/pages/${id}`);
}

// Public page retrieval (for display on public pages)
export async function getPageBySlug(slug: string): Promise<Page | null> {
  try {
    // Public endpoint - skip auth
    return await apiGet<Page>(`/pages/${slug}`, { skipAuth: true });
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error(`Error fetching page by slug ${slug}:`, error);
    return null;
  }
}

// Helper functions for specific pages (for backward compatibility with existing about/contact pages)
export async function getAboutPage() {
  return getPageBySlug('about-me');
}

export async function getContactPage() {
  return getPageBySlug('contact-me');
}

