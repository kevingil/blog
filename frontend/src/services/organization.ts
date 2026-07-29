import { Organizations } from '@/client';
import { generatedData } from '@/services/generatedClient';

/**
 * Organization data
 */
export interface Organization {
  id: string;
  name: string;
  slug: string;
  bio: string;
  logo_url: string;
  website_url: string;
  email_public: string;
  social_links: Record<string, string>;
  meta_description: string;
}

/**
 * Request to create an organization
 */
export interface OrganizationCreateRequest {
  name: string;
  slug?: string;
  bio?: string;
  logo_url?: string;
  website_url?: string;
  email_public?: string;
  social_links?: Record<string, string>;
  meta_description?: string;
}

/**
 * Request to update an organization
 */
export interface OrganizationUpdateRequest {
  name?: string;
  slug?: string;
  bio?: string;
  logo_url?: string;
  website_url?: string;
  email_public?: string;
  social_links?: Record<string, string>;
  meta_description?: string;
}

/**
 * List all organizations
 */
export async function listOrganizations(): Promise<Organization[]> {
  try {
    return await generatedData<Organization[]>(Organizations.listOrganizations());
  } catch (error: any) {
    console.error('Error fetching organizations:', error);
    return [];
  }
}

/**
 * Get an organization by ID
 */
export async function getOrganization(id: string): Promise<Organization | null> {
  try {
    return await generatedData<Organization>(Organizations.getOrganization({ path: { id } }));
  } catch (error: any) {
    if (error.status === 404) {
      return null;
    }
    console.error('Error fetching organization:', error);
    return null;
  }
}

/**
 * Create a new organization
 */
export async function createOrganization(data: OrganizationCreateRequest): Promise<Organization> {
  return generatedData<Organization>(Organizations.createOrganization({ body: data }));
}

/**
 * Update an organization
 */
export async function updateOrganization(id: string, data: OrganizationUpdateRequest): Promise<Organization> {
  return generatedData<Organization>(Organizations.updateOrganization({ path: { id }, body: data }));
}

/**
 * Delete an organization
 */
export async function deleteOrganization(id: string): Promise<boolean> {
  try {
    await generatedData<{ success: boolean }>(Organizations.deleteOrganization({ path: { id } }));
    return true;
  } catch (error: any) {
    console.error('Error deleting organization:', error);
    throw error;
  }
}

/**
 * Join an organization
 */
export async function joinOrganization(orgId: string): Promise<boolean> {
  try {
    await generatedData<{ success: boolean }>(Organizations.joinOrganization({ path: { id: orgId } }));
    return true;
  } catch (error: any) {
    console.error('Error joining organization:', error);
    throw error;
  }
}

/**
 * Leave current organization
 */
export async function leaveOrganization(): Promise<boolean> {
  try {
    await generatedData<{ success: boolean }>(Organizations.leaveOrganization());
    return true;
  } catch (error: any) {
    console.error('Error leaving organization:', error);
    throw error;
  }
}
