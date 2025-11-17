import { VITE_API_BASE_URL } from "../constants";
import { getAuthHeaders, getAuthHeadersWithContentType } from "../auth/utils";
import { handleApiResponse } from "../apiHelpers";

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

declare const process: {
    env: {
        NEXT_PUBLIC_API_URL?: string;
    };
};

export async function listFiles(prefix: string | null): Promise<{ files: FileData[], folders: FolderData[] }> {
    const url = `${VITE_API_BASE_URL}/storage/files${prefix ? `?prefix=${encodeURIComponent(prefix)}` : ''}`;
    const response = await fetch(url, {
        headers: getAuthHeaders(),
    });

    return handleApiResponse<{ files: FileData[], folders: FolderData[] }>(response);
}

export async function uploadFile(key: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('key', key);

    const response = await fetch(`${VITE_API_BASE_URL}/storage/upload`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: formData,
    });

    return handleApiResponse<any>(response);
}

export async function deleteFile(key: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/${encodeURIComponent(key)}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
    });

    await handleApiResponse<{ success: boolean }>(response);
}

export async function createFolder(folderPath: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/folders`, {
        method: 'POST',
        headers: getAuthHeadersWithContentType(),
        body: JSON.stringify({ path: folderPath }),
    });

    await handleApiResponse<{ success: boolean }>(response);
}

export async function updateFolder(oldPath: string, newPath: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/folders`, {
        method: 'PUT',
        headers: getAuthHeadersWithContentType(),
        body: JSON.stringify({ oldPath, newPath }),
    });

    await handleApiResponse<{ success: boolean }>(response);
}

