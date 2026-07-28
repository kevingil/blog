import { Projects } from '@/client';
import { generatedData } from '@/services/generatedClient';

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
  return generatedData<ListProjectsResponse>(
    Projects.listProjects({ query: { page, perPage } }),
  );
}

export async function getProject(id: string): Promise<ProjectDetail> {
  // Public endpoint - skip auth
  return generatedData<ProjectDetail>(Projects.getProject({ path: { id } }));
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
  return generatedData<Project>(Projects.createProject({ body: payload }));
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
  return generatedData<Project>(Projects.updateProject({ path: { id }, body: payload }));
}

export async function deleteProject(id: string): Promise<{ success: boolean }> {
  // Protected endpoint - requires auth
  return generatedData<{ success: boolean }>(Projects.deleteProject({ path: { id } }));
}

