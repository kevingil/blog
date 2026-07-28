import { apiGet, apiPost, apiPut, apiDelete } from '@/services/authenticatedFetch';

export type Project = {
  id: string;
  title: string;
  description: string;
  content?: string;
  tag_ids?: number[];
  image_url?: string;
  url?: string;
  created_at: string;
  updated_at: string;
};

export type ProjectDetail = {
  project: Project;
  tags: string[];
}

export type ListProjectsResponse = {
  projects: Project[];
  total: number;
  current_page: number;
  per_page: number;
};

export async function listProjects(page: number = 1, perPage: number = 20): Promise<ListProjectsResponse> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) });
  // Public endpoint - skip auth
  return apiGet<ListProjectsResponse>(`/projects/?${params}`, { skipAuth: true });
}

export async function getProject(id: string): Promise<ProjectDetail> {
  // Public endpoint - skip auth
  return apiGet<ProjectDetail>(`/projects/${id}`, { skipAuth: true });
}

export async function createProject(payload: {
  title: string;
  description: string;
  content?: string;
  tags?: string[];
  image_url?: string;
  url?: string;
}): Promise<Project> {
  // Protected endpoint - requires auth
  return apiPost<Project>('/projects/', payload);
}

export async function updateProject(id: string, payload: {
  title?: string;
  description?: string;
  content?: string;
  tags?: string[];
  image_url?: string;
  url?: string;
  created_at?: string;
}): Promise<Project> {
  // Protected endpoint - requires auth
  return apiPut<Project>(`/projects/${id}`, payload);
}

export async function deleteProject(id: string): Promise<{ success: boolean }> {
  // Protected endpoint - requires auth
  return apiDelete<{ success: boolean }>(`/projects/${id}`);
}


