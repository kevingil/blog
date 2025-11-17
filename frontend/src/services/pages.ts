import { VITE_API_BASE_URL } from '@/services/constants';
import { handleApiResponse } from '@/services/apiHelpers';
import { getAuthHeadersWithContentType } from './auth/utils';

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

  const response = await fetch(`${VITE_API_BASE_URL}/dashboard/pages?${params}`, {
    headers: getAuthHeadersWithContentType(),
  });

  return handleApiResponse<PageListResponse>(response);
}

export async function getPage(id: string): Promise<Page> {
  const response = await fetch(`${VITE_API_BASE_URL}/dashboard/pages/${id}`, {
    headers: getAuthHeadersWithContentType(),
  });

  return handleApiResponse<Page>(response);
}

export async function createPage(data: PageCreateRequest): Promise<Page> {
  const response = await fetch(`${VITE_API_BASE_URL}/dashboard/pages`, {
    method: 'POST',
    headers: getAuthHeadersWithContentType(),
    body: JSON.stringify(data),
  });

  return handleApiResponse<Page>(response);
}

export async function updatePage(id: string, data: PageUpdateRequest): Promise<Page> {
  const response = await fetch(`${VITE_API_BASE_URL}/dashboard/pages/${id}`, {
    method: 'PUT',
    headers: getAuthHeadersWithContentType(),
    body: JSON.stringify(data),
  });

  return handleApiResponse<Page>(response);
}

export async function deletePage(id: string): Promise<{ success: boolean }> {
  const response = await fetch(`${VITE_API_BASE_URL}/dashboard/pages/${id}`, {
    method: 'DELETE',
    headers: getAuthHeadersWithContentType(),
  });

  return handleApiResponse<{ success: boolean }>(response);
}

// Public page retrieval (for display on public pages)
export async function getPageBySlug(slug: string): Promise<Page | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/pages/${slug}`);
    
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch page');
    }
    
    return handleApiResponse<Page>(response);
  } catch (error) {
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

