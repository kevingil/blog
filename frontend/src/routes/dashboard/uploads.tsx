import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { Card, CardContent } from '../../components/ui/card';
import { isApiError } from '../../services/authenticatedFetch';
import { createFileRoute } from '@tanstack/react-router';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { SortingState } from '@tanstack/react-table';
import { DataTable } from '@/components/uploads/data-table/data-table';
import { createColumns, UploadItem } from '@/components/uploads/data-table/columns';
import { FileGrid } from '@/components/uploads/file-grid';
import { ViewMode } from '@/components/uploads/data-table/data-table-toolbar';
import { useToast } from '@/hooks/use-toast';
import {
  useStorageFiles,
  useStorageUploadMutation,
  useStorageDeleteMutation,
  useStorageCreateFolderMutation,
} from '@/hooks/use-storage-files';
import { Upload } from 'lucide-react';

export const Route = createFileRoute('/dashboard/uploads')({
  component: UploadsPage,
});

const LOG_PREFIX = '[Uploads]';

function UploadsPage() {
  const [currentPath, setCurrentPath] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('');
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'name', desc: false }
  ]);
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [isDraggingOver, setIsDraggingOver] = useState(false);
  const dragCounterRef = useRef(0);

  const { setPageTitle } = useAdminDashboard();
  const { toast } = useToast();

  const { data, isLoading, error } = useStorageFiles(currentPath);
  const uploadMutation = useStorageUploadMutation();
  const deleteMutation = useStorageDeleteMutation();
  const createFolderMutation = useStorageCreateFolderMutation();

  const files = data?.files ?? [];
  const folders = data?.folders ?? [];
  const fetchingError = error ? (error instanceof Error ? error.message : 'An unknown error occurred') : null;
  const isUploading = uploadMutation.isPending;

  useEffect(() => {
    setPageTitle("Uploads");
  }, [setPageTitle]);

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const combinedData: UploadItem[] = useMemo(() => {
    const folderItems: UploadItem[] = folders.map(folder => ({
      ...folder,
      type: 'folder' as const,
    }));

    const fileItems: UploadItem[] = files.map(file => ({
      ...file,
      type: 'file' as const,
    }));

    const filteredItems = [...folderItems, ...fileItems].filter(item => {
      if (!debouncedSearchQuery) return true;

      const searchLower = debouncedSearchQuery.toLowerCase();
      if (item.type === 'folder') {
        return item.name.toLowerCase().includes(searchLower);
      } else {
        const fileName = item.key.split('/').pop() || item.key;
        return fileName.toLowerCase().includes(searchLower);
      }
    });

    return filteredItems;
  }, [files, folders, debouncedSearchQuery]);

  // ── Multi-file upload with logging ──

  const uploadFiles = useCallback(async (fileList: File[], source: string) => {
    if (fileList.length === 0) {
      console.warn(`${LOG_PREFIX} uploadFiles called with empty list from ${source}`);
      return;
    }

    console.log(`${LOG_PREFIX} uploadFiles start: ${fileList.length} file(s) from ${source}, targetPath="${currentPath}"`);
    fileList.forEach((f, i) => {
      console.log(`${LOG_PREFIX}   [${i}] name="${f.name}" type="${f.type}" size=${f.size}`);
    });

    toast({
      title: `Uploading ${fileList.length} file${fileList.length > 1 ? 's' : ''}...`,
      description: `To ${currentPath || '/'}`,
    });

    const results = await Promise.allSettled(
      fileList.map(async (file) => {
        const key = `${currentPath}${file.name}`;
        console.log(`${LOG_PREFIX} uploading "${file.name}" as key="${key}" (${file.size} bytes)`);
        try {
          await uploadMutation.mutateAsync({ key, file });
          console.log(`${LOG_PREFIX} upload success: "${file.name}" -> key="${key}"`);
          return { name: file.name, key };
        } catch (err) {
          console.error(`${LOG_PREFIX} upload failed: "${file.name}" -> key="${key}"`, err);
          throw err;
        }
      })
    );

    const succeeded = results.filter(r => r.status === 'fulfilled');
    const failed = results.filter(r => r.status === 'rejected') as PromiseRejectedResult[];

    const getErrorMessage = (err: unknown): string =>
      isApiError(err) ? err.message : err instanceof Error ? err.message : 'Unknown error';

    const failedMessages = failed.map(f => getErrorMessage(f.reason));
    const errorDetail = failedMessages.length > 0 ? failedMessages[0] : '';

    console.log(`${LOG_PREFIX} uploadFiles done: ${succeeded.length} succeeded, ${failed.length} failed`, failedMessages);

    if (failed.length === 0) {
      toast({
        title: `Uploaded ${succeeded.length} file${succeeded.length > 1 ? 's' : ''}`,
        description: succeeded.length === 1
          ? (succeeded[0] as PromiseFulfilledResult<{ name: string }>).value.name
          : `All files uploaded to ${currentPath || '/'}`,
      });
    } else if (succeeded.length === 0) {
      toast({
        title: `Upload failed`,
        description: errorDetail || `All ${failed.length} file${failed.length > 1 ? 's' : ''} failed to upload`,
        variant: 'destructive',
      });
    } else {
      toast({
        title: `Upload partially completed`,
        description: `${succeeded.length} succeeded, ${failed.length} failed${errorDetail ? `: ${errorDetail}` : ''}`,
        variant: 'destructive',
      });
    }
  }, [currentPath, toast, uploadMutation]);

  // ── Input-based upload (now supports multiple) ──

  const handleFileUpload = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const inputFiles = event.target.files;
    if (!inputFiles || inputFiles.length === 0) {
      console.log(`${LOG_PREFIX} handleFileUpload: no files selected`);
      return;
    }
    console.log(`${LOG_PREFIX} handleFileUpload: ${inputFiles.length} file(s) selected via input`);
    await uploadFiles(Array.from(inputFiles), 'file-input');
    event.target.value = '';
  }, [uploadFiles]);

  // ── Drag and drop handlers ──

  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current += 1;
    console.log(`${LOG_PREFIX} dragEnter, counter=${dragCounterRef.current}, types=`, e.dataTransfer.types);
    if (e.dataTransfer.types.includes('Files')) {
      setIsDraggingOver(true);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current -= 1;
    console.log(`${LOG_PREFIX} dragLeave, counter=${dragCounterRef.current}`);
    if (dragCounterRef.current <= 0) {
      dragCounterRef.current = 0;
      setIsDraggingOver(false);
    }
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current = 0;
    setIsDraggingOver(false);

    const droppedFiles = Array.from(e.dataTransfer.files);
    console.log(`${LOG_PREFIX} drop: ${droppedFiles.length} file(s) dropped`);

    if (droppedFiles.length === 0) {
      console.warn(`${LOG_PREFIX} drop: dataTransfer had no files`);
      return;
    }

    await uploadFiles(droppedFiles, 'drag-and-drop');
  }, [uploadFiles]);

  // ── CRUD operations ──

  const handleDeleteFile = useCallback(async (key: string) => {
    console.log(`${LOG_PREFIX} deleteFile: key="${key}"`);
    try {
      await deleteMutation.mutateAsync(key);
      console.log(`${LOG_PREFIX} deleteFile success: key="${key}"`);
    } catch (error) {
      const msg = isApiError(error) ? error.message : error instanceof Error ? error.message : 'Failed to delete file';
      console.error(`${LOG_PREFIX} deleteFile error: key="${key}"`, error);
      toast({ title: 'Delete failed', description: msg, variant: 'destructive' });
    }
  }, [deleteMutation, toast]);

  const handleCreateFolder = useCallback(async (folderName: string) => {
    if (!folderName) return;
    const path = `${currentPath}${folderName}/`;
    console.log(`${LOG_PREFIX} createFolder: path="${path}"`);
    try {
      await createFolderMutation.mutateAsync(path);
      console.log(`${LOG_PREFIX} createFolder success: path="${path}"`);
    } catch (error) {
      const msg = isApiError(error) ? error.message : error instanceof Error ? error.message : 'Failed to create folder';
      console.error(`${LOG_PREFIX} createFolder error: path="${path}"`, error);
      toast({ title: 'Create folder failed', description: msg, variant: 'destructive' });
    }
  }, [currentPath, createFolderMutation, toast]);

  const navigateToPath = useCallback((path: string) => {
    console.log(`${LOG_PREFIX} navigateToPath: "${path}"`);
    setCurrentPath(path);
  }, []);

  const columns = useMemo(
    () => createColumns({
      onDelete: handleDeleteFile,
      onFolderClick: navigateToPath,
    }),
    [handleDeleteFile, navigateToPath]
  );

  const isEmpty = !isLoading && combinedData.length === 0;

  return (
    <div
      className="flex flex-col flex-1 p-0 md:p-4 mb-4 h-full overflow-hidden relative"
      onDragEnter={handleDragEnter}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {fetchingError && (
        <Card className='border-red-200 dark:border-red-700 p-2 text-gray-800 dark:text-white mb-4'>
          <CardContent>
            <p className='text-gray-800 dark:text-white font-medium text-sm'>
              <span className='font-medium text-red-500'>Error:</span> {fetchingError}
            </p>
          </CardContent>
        </Card>
      )}

      <DataTable
        columns={columns}
        data={combinedData}
        onSortingChange={setSorting}
        sorting={sorting}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        isLoading={isLoading}
        currentPath={currentPath}
        onNavigateToPath={navigateToPath}
        onFileUpload={handleFileUpload}
        onCreateFolder={handleCreateFolder}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        gridView={
          isEmpty ? (
            <div className="flex flex-col items-center justify-center h-48 text-muted-foreground gap-3 border rounded-md border-dashed">
              <Upload className="h-10 w-10" />
              <p className="text-sm">Drop files here or click Upload</p>
              <p className="text-xs">Files will be uploaded to <span className="font-mono">{currentPath || '/'}</span></p>
            </div>
          ) : (
            <FileGrid
              data={combinedData}
              onDelete={handleDeleteFile}
              onFolderClick={navigateToPath}
              isLoading={isLoading}
            />
          )
        }
      />

      {/* Drag overlay */}
      {isDraggingOver && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm border-2 border-dashed border-primary rounded-lg pointer-events-none">
          <div className="flex flex-col items-center gap-3 text-primary">
            <Upload className="h-16 w-16 animate-bounce" />
            <p className="text-lg font-medium">
              Drop to upload to <span className="font-mono">{currentPath || '/'}</span>
            </p>
            {isUploading && (
              <p className="text-sm text-muted-foreground">Uploading...</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
