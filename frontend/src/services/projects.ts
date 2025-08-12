import { VITE_API_BASE_URL } from '@/services/constants';

export type Project = {
  id: string;
  title: string;
  description: string;
  image_url?: string;
  url?: string;
  created_at: string;
  updated_at: string;
};

export type ListProjectsResponse = {
  projects: Project[];
  total: number;
  current_page: number;
  per_page: number;
};

export async function listProjects(page: number = 1, perPage: number = 20): Promise<ListProjectsResponse> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) });
  const res = await fetch(`${VITE_API_BASE_URL}/projects/?${params}`);
  if (!res.ok) throw new Error('Failed to list projects');
  return res.json();
}

export async function getProject(id: string): Promise<Project> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`);
  if (!res.ok) throw new Error('Failed to fetch project');
  return res.json();
}

export async function createProject(payload: {
  title: string;
  description: string;
  image_url?: string;
  url?: string;
}): Promise<Project> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to create project');
  return res.json();
}

export async function updateProject(id: string, payload: {
  title?: string;
  description?: string;
  image_url?: string;
  url?: string;
}): Promise<Project> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to update project');
  return res.json();
}

export async function deleteProject(id: string): Promise<{ success: boolean }> {
  const res = await fetch(`${VITE_API_BASE_URL}/projects/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete project');
  return res.json();
}


