import { API_BASE_URL } from "../constants";

export type FileData = {
    key: string;
    lastModified: Date;
    size: string;
    sizeRaw: number;
    url: string;
    isImage: boolean;
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
    const response = await fetch(`${API_BASE_URL}/api/storage/list?prefix=${encodeURIComponent(prefix || '')}`);
    if (!response.ok) {
        throw new Error('Failed to list files');
    }
    return response.json();
}

export async function uploadFile(key: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('key', key);

    const response = await fetch(`${API_BASE_URL}/api/storage/upload`, {
        method: 'POST',
        body: formData,
    });

    if (!response.ok) {
        throw new Error('Failed to upload file');
    }
    return response.json();
}

export async function deleteFile(key: string) {
    const response = await fetch(`${API_BASE_URL}/api/storage/delete`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ key }),
    });

    if (!response.ok) {
        throw new Error('Failed to delete file');
    }
}

export async function createFolder(folderPath: string) {
    const response = await fetch(`${API_BASE_URL}/api/storage/folder`, {
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
    const response = await fetch(`${API_BASE_URL}/api/storage/folder`, {
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

