import { apiGet, apiPut } from '@/services/authenticatedFetch';

/**
 * Public profile that can be either a user or organization
 */
export interface PublicProfile {
  type: 'user' | 'organization';
  id: string;
  name: string;
  bio: string;
  image_url: string; // profile_image for user, logo_url for org
  email_public: string;
  social_links: Record<string, string>;
  meta_description: string;
  website_url?: string; // org only
}

/**
 * User profile data
 */
export interface UserProfile {
  id: string;
  name: string;
  bio: string;
  profile_image: string;
  email_public: string;
  social_links: Record<string, string>;
  meta_description: string;
  organization_id?: string;
}

/**
 * Request to update user profile
 */
export interface ProfileUpdateRequest {
  name?: string;
  bio?: string;
  profile_image?: string;
  email_public?: string;
  social_links?: Record<string, string>;
  meta_description?: string;
}

/**
 * Site settings for controlling public profile
 */
export interface SiteSettings {
  public_profile_type: 'user' | 'organization';
  public_user_id: string | null;
  public_organization_id: string | null;
}

/**
 * Request to update site settings
 */
export interface SiteSettingsUpdateRequest {
  public_profile_type?: 'user' | 'organization';
  public_user_id?: string;
  public_organization_id?: string;
}

/**
 * Get the public profile based on site settings
 * This is what shows on the /about page
 */
export async function getPublicProfile(): Promise<PublicProfile | null> {
  try {
    return await apiGet<PublicProfile>('/profile/public', { skipAuth: true });
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error('Error fetching public profile:', error);
    return null;
  }
}

/**
 * Get the authenticated user's profile
 */
export async function getMyProfile(): Promise<UserProfile | null> {
  try {
    return await apiGet<UserProfile>('/profile');
  } catch (error: any) {
    console.error('Error fetching my profile:', error);
    return null;
  }
}

/**
 * Update the authenticated user's profile
 */
export async function updateProfile(data: ProfileUpdateRequest): Promise<UserProfile | null> {
  try {
    return await apiPut<UserProfile>('/profile', data);
  } catch (error: any) {
    console.error('Error updating profile:', error);
    throw error;
  }
}

/**
 * Get current site settings
 */
export async function getSiteSettings(): Promise<SiteSettings | null> {
  try {
    return await apiGet<SiteSettings>('/profile/settings');
  } catch (error: any) {
    console.error('Error fetching site settings:', error);
    return null;
  }
}

/**
 * Update site settings
 */
export async function updateSiteSettings(data: SiteSettingsUpdateRequest): Promise<SiteSettings | null> {
  try {
    return await apiPut<SiteSettings>('/profile/settings', data);
  } catch (error: any) {
    console.error('Error updating site settings:', error);
    throw error;
  }
}
