import { apiGet, apiPut } from '@/services/authenticatedFetch';

export interface User {
  id: number;
  name: string;
  email: string;
  role: string;
  created_at: number;
  updated_at: number;
}

export interface ContactPageData {
  id: number;
  title: string;
  content: string;
  email_address: string;
  social_links?: string;
  meta_description?: string;
  last_updated: string;
  description?: string;
}

export interface AboutPageData {
  id: number;
  title: string;
  content: string;
  profile_image?: string;
  meta_description?: string;
  last_updated: string;
  description?: string;
}

export async function getUser(): Promise<User | null> {
  // In a real app, this would fetch the user from your backend
  // For now, we'll return null to indicate no user is logged in
  return null;
} 

export async function updateContactPage(data: ContactPageData): Promise<ContactPageData | null> {
  // Protected endpoint - requires auth
  return apiPut<ContactPageData>('/pages/contact-me', data);
}

export async function updateAboutPage(data: AboutPageData): Promise<AboutPageData | null> {
  // Protected endpoint - requires auth
  return apiPut<AboutPageData>('/pages/about-me', data);
}

export async function getContactPage(): Promise<ContactPageData | null> {
  try {
    // Public endpoint - skip auth
    const page = await apiGet<any>('/pages/contact-me', { skipAuth: true });
    
    // meta_data is a JSON object, parse social_links and email_address from it
    let meta: Record<string, any> = {};
    try {
      meta = typeof page.meta_data === 'string' ? JSON.parse(page.meta_data) : page.meta_data || {};
    } catch (e) {
      meta = {};
    }
    return {
      id: page.id,
      title: page.title,
      content: page.content || '',
      description: page.description || '',
      email_address: meta.email_address || '',
      social_links: meta.social_links || '{}',
      meta_description: '',
      last_updated: page.updated_at || page.created_at || '',
    };
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error('Error fetching contact page:', error);
    return null;
  }
}

export async function getAboutPage(): Promise<AboutPageData | null> {
  try {
    // Public endpoint - skip auth
    const page = await apiGet<any>('/pages/about-me', { skipAuth: true });
    
    return {
      id: page.id,
      title: page.title,
      content: page.content || '',
      description: page.description || '',
      profile_image: page.image_url || '',
      meta_description: '',
      last_updated: page.updated_at || page.created_at || '',
    };
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error('Error fetching about page:', error);
    return null;
  }
}
