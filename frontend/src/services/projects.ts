import { VITE_API_BASE_URL } from '@/services/constants';
import { handleApiResponse } from '@/services/apiHelpers';

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
  const res = await fetch(`${VITE_API_BASE_URL}/projects/?${params}`);
  return handleApiResponse<ListProjectsResponse>(res);
}

export async function getProject(id: string): Promise<ProjectDetail> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`);
  return handleApiResponse<ProjectDetail>(res);
}

export async function createProject(payload: {
  title: string;
  description: string;
  content?: string;
  tags?: string[];
  image_url?: string;
  url?: string;
}): Promise<Project> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return handleApiResponse<Project>(res);
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
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return handleApiResponse<Project>(res);
}

export async function deleteProject(id: string): Promise<{ success: boolean }> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`, { method: 'DELETE' });
  return handleApiResponse<{ success: boolean }>(res);
}


