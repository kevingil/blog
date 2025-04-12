'use server'

import { S3Client, ListObjectsV2Command, PutObjectCommand, DeleteObjectCommand, CopyObjectCommand } from "@aws-sdk/client-s3";

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

const bucket = process.env.S3_BUCKET;


const s3Client = new S3Client({
    region: "auto",
    endpoint: process.env.S3_ENDPOINT,
    credentials: {
        accessKeyId: process.env.S3_ACCESS_KEY_ID!,
        secretAccessKey: process.env.S3_ACCESS_KEY_SECRET!,
    },
});

export async function listFiles(prefix: string | null): Promise<{ files: FileData[], folders: FolderData[] }> {
    let response;
    try {
        const command = new ListObjectsV2Command({
            Prefix: prefix || "",
            Bucket: bucket,
        });

        response = await s3Client.send(command);

    } catch (error) {
        throw error as Error;
    }


    const files: FileData[] = response?.Contents?.map(item => ({
        key: item.Key!,
        lastModified: item.LastModified!,
        size: formatByteSize(item.Size!),
        sizeRaw: item.Size!,
        url: `${process.env.NEXT_PUBLIC_S3_URL_PREFIX}/${item.Key}`,
        isImage: isImageFile(item.Key!),
    })) || [];
    files.sort((a, b) => b.lastModified.getTime() - a.lastModified.getTime());

    const folders: FolderData[] = response?.CommonPrefixes?.map(prefix => ({
        name: prefix.Prefix!.split("/").slice(-2)[0],
        path: prefix.Prefix!,
        isHidden: folderIsHidden(prefix.Prefix!.split("/").slice(-2)[0]),
        lastModified: new Date(),
        fileCount: 0,
    })) || [];

    return { files, folders };
}

export async function uploadFile(key: string, file: File) {
    const body = Buffer.from( await file.arrayBuffer());
    const command = new PutObjectCommand({
        Bucket: bucket,
        Key: key,
        Body: body
    });
    try {
        const response = await s3Client.send(command);
        console.log("uploadFile response", response);
        return response;
    } catch (error) {
        throw error as Error;
    }
}

export async function deleteFile(key: string) {
    const command = new DeleteObjectCommand({
        Bucket: bucket,
        Key: key,
    });

    await s3Client.send(command);
}

export async function createFolder(folderPath: string) {
    const command = new PutObjectCommand({
        Bucket: bucket,
        Key: folderPath.endsWith("/") ? folderPath : `${folderPath}/`,
        Body: Buffer.from(""),
    });

    await s3Client.send(command);
}

export async function updateFolder(oldPath: string, newPath: string) {
    const listCommand = new ListObjectsV2Command({
        Bucket: bucket,
        Prefix: oldPath,
    });

    const listResponse = await s3Client.send(listCommand);

    for (const object of listResponse.Contents || []) {
        const newKey = object.Key!.replace(oldPath, newPath);

        const copyCommand = new CopyObjectCommand({
            Bucket: bucket,
            CopySource: `${bucket}/${object.Key}`,
            Key: newKey,
        });

        await s3Client.send(copyCommand);

        const deleteCommand = new DeleteObjectCommand({
            Bucket: bucket,
            Key: object.Key!,
        });

        await s3Client.send(deleteCommand);
    }
}

function formatByteSize(size: number): string {
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    let i = 0;
    while (size >= 1024 && i < units.length - 1) {
        size /= 1024;
        i++;
    }
    return `${size.toFixed(2)} ${units[i]}`;
}

function isImageFile(key: string): boolean {
    const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp'];
    const ext = key.split('.').pop()?.toLowerCase();
    return imageExtensions.includes(`.${ext}`);
}

function folderIsHidden(folderName: string): boolean {
    return folderName.startsWith('.');
}
