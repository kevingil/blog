import { VITE_API_BASE_URL } from '@/services/constants';

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
}


export interface AboutPageData {
  id: number;
  title: string;
  content: string;
  profile_image?: string;
  meta_description?: string;
  last_updated: string;
}

export async function getUser(): Promise<User | null> {
  // In a real app, this would fetch the user from your backend
  // For now, we'll return null to indicate no user is logged in
  return null;
} 

export async function updateContactPage(data: ContactPageData): Promise<ContactPageData | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/pages/contact`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return await response.json();
}

export async function updateAboutPage(data: AboutPageData): Promise<AboutPageData | null> {
  const response = await fetch(`${VITE_API_BASE_URL}/pages/about`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  return await response.json();
}





export async function getContactPage(): Promise<ContactPageData | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/pages/contact`);
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch contact page');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching contact page:', error);
    return null;
  }
}

export async function getAboutPage(): Promise<AboutPageData | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/pages/about`);
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error('Failed to fetch about page');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching about page:', error);
    return null;
  }
}
