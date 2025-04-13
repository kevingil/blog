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
  emailAddress: string;
  socialLinks?: string;
  metaDescription?: string;
  lastUpdated: string;
}


export interface AboutPageData {
  id: number;
  title: string;
  content: string;
  profileImage?: string;
  metaDescription?: string;
  lastUpdated: string;
}

export async function getUser(): Promise<User | null> {
  // In a real app, this would fetch the user from your backend
  // For now, we'll return null to indicate no user is logged in
  return null;
} 

export async function getContactPage(): Promise<ContactPageData | null> {
  // In a real app, this would fetch the contact page from your backend
  // For now, we'll return null to indicate no contact page is available
  return null;
}

export async function getAboutPage(): Promise<AboutPageData | null> {
  // In a real app, this would fetch the about page from your backend
  // For now, we'll return null to indicate no about page is available
  return null;
}
