import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listFiles,
  uploadFile as uploadFileApi,
  deleteFile as deleteFileApi,
  createFolder as createFolderApi,
} from '@/services/storage';

const STORAGE_QUERY_KEY = ['storage', 'files'] as const;

export function useStorageFiles(prefix: string) {
  return useQuery({
    queryKey: [...STORAGE_QUERY_KEY, prefix],
    queryFn: () => listFiles(prefix || null),
  });
}

export function useStorageUploadMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ key, file }: { key: string; file: File }) => {
      return uploadFileApi(key, file);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: STORAGE_QUERY_KEY });
    },
  });
}

export function useStorageDeleteMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => deleteFileApi(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: STORAGE_QUERY_KEY });
    },
  });
}

export function useStorageCreateFolderMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (path: string) => createFolderApi(path),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: STORAGE_QUERY_KEY });
    },
  });
}
