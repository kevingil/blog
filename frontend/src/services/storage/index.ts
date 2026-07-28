import { VITE_API_BASE_URL } from "../constants";
import { apiGet, apiPost, apiPut, apiDelete, authenticatedFetch } from "../authenticatedFetch";

export type FileData = {
    key: string;
    last_modified: Date;
    size: string;
    size_raw: number;
    url: string;
    is_image: boolean;
};

export type FolderData = {
    name: string;
    path: string;
    isHidden: boolean;
    lastModified: Date;
    fileCount: number;
};

export async function listFiles(prefix: string | null): Promise<{ files: FileData[], folders: FolderData[] }> {
    const url = `/storage/files${prefix ? `?prefix=${encodeURIComponent(prefix)}` : ''}`;
    return apiGet<{ files: FileData[], folders: FolderData[] }>(url);
}

export async function uploadFile(key: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('key', key);

    // For FormData, we need to use authenticatedFetch directly without Content-Type
    return authenticatedFetch<any>(`${VITE_API_BASE_URL}/storage/upload`, {
        method: 'POST',
        body: formData,
        headers: {}, // Let browser set Content-Type with boundary for FormData
    });
}

export async function deleteFile(key: string) {
    await apiDelete<{ success: boolean }>(`/storage/${encodeURIComponent(key)}`);
}

export async function createFolder(folderPath: string) {
    await apiPost<{ success: boolean }>('/storage/folders', { path: folderPath });
}

export async function updateFolder(oldPath: string, newPath: string) {
    await apiPut<{ success: boolean }>('/storage/folders', { oldPath, newPath });
}

