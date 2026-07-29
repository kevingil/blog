import { Storage } from "@/client";
import type {
    FileDataResponse,
    FolderDataResponse,
    ListFilesResponse,
} from "@/client";
import { generatedData } from "../generatedClient";

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
    const data = await generatedData<ListFilesResponse>(
        Storage.listStorageFiles({ query: { prefix: prefix ?? undefined } }),
    );
    return {
        files: data.files.map(toFileData),
        folders: data.folders.map(toFolderData),
    };
}

export async function uploadFile(key: string, file: File) {
    return generatedData(
        Storage.uploadStorageFile({ body: { key, file } }),
    );
}

export async function deleteFile(key: string) {
    await generatedData<{ success: boolean }>(
        Storage.deleteStorageFile({ path: { key } }),
    );
}

export async function createFolder(folderPath: string) {
    await generatedData<{ success: boolean }>(
        Storage.createStorageFolder({ body: { path: folderPath } }),
    );
}

export async function updateFolder(oldPath: string, newPath: string) {
    await generatedData<{ success: boolean }>(
        Storage.updateStorageFolder({ body: { oldPath, newPath } }),
    );
}

function toFileData(file: FileDataResponse): FileData {
    return { ...file, last_modified: new Date(file.last_modified) };
}

function toFolderData(folder: FolderDataResponse): FolderData {
    return {
        name: folder.name,
        path: folder.path,
        isHidden: folder.is_hidden,
        lastModified: new Date(folder.last_modified),
        fileCount: folder.file_count,
    };
}
