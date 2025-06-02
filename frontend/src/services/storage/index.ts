import { VITE_API_BASE_URL } from "../constants";

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
    const response = await fetch(url);

    if (!response.ok) {
        const errorText = await response.text();
        const errorMessage = JSON.parse(errorText);
        console.log('Error message:', errorMessage.error);
        throw new Error(errorMessage.error);
    }
    return response.json();
}

export async function uploadFile(key: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('key', key);

    const response = await fetch(`${VITE_API_BASE_URL}/storage/upload`, {
        method: 'POST',
        body: formData,
    });

    if (!response.ok) {
        throw new Error('Failed to upload file');
    }
    return response.json();
}

export async function deleteFile(key: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/${encodeURIComponent(key)}`, {
        method: 'DELETE',
    });

    if (!response.ok) {
        throw new Error('Failed to delete file');
    }
}

export async function createFolder(folderPath: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/folders`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ path: folderPath }),
    });

    if (!response.ok) {
        throw new Error('Failed to create folder');
    }
}

export async function updateFolder(oldPath: string, newPath: string) {
    const response = await fetch(`${VITE_API_BASE_URL}/storage/folders`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ oldPath, newPath }),
    });

    if (!response.ok) {
        throw new Error('Failed to update folder');
    }
}

