import { VITE_API_BASE_URL } from '@/services/constants';
import { handleApiResponse } from '@/services/apiHelpers';

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
  const response = await fetch(`${VITE_API_BASE_URL}/pages/contact-me`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return handleApiResponse<ContactPageData>(response);
}

export async function updateAboutPage(data: AboutPageData): Promise<AboutPageData | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/pages/about-me`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return handleApiResponse<AboutPageData>(response);
}





export async function getContactPage(): Promise<ContactPageData | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/pages/contact-me`);
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch contact page');
    }
    const page = await handleApiResponse<any>(response);
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
  } catch (error) {
    console.error('Error fetching contact page:', error);
    return null;
  }
}

export async function getAboutPage(): Promise<AboutPageData | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/pages/about-me`);
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch about page');
    }
    const page = await handleApiResponse<any>(response);
    return {
      id: page.id,
      title: page.title,
      content: page.content || '',
      description: page.description || '',
      profile_image: page.image_url || '',
      meta_description: '',
      last_updated: page.updated_at || page.created_at || '',
    };
  } catch (error) {
    console.error('Error fetching about page:', error);
    return null;
  }
}
