import { useState, useEffect, useMemo } from 'react';
import { Card, CardContent } from '../../components/ui/card';
import { listFiles, uploadFile, deleteFile, createFolder, FileData, FolderData } from '../../services/storage';
import { createFileRoute } from '@tanstack/react-router';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { SortingState } from '@tanstack/react-table';
import { DataTable } from '@/components/uploads/data-table/data-table';
import { createColumns, UploadItem } from '@/components/uploads/data-table/columns';

export const Route = createFileRoute('/dashboard/uploads')({
  component: UploadsPage,
});

function UploadsPage() {
  const [files, setFiles] = useState<FileData[]>([]);
  const [folders, setFolders] = useState<FolderData[]>([]);
  const [currentPath, setCurrentPath] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('');
  const [fetchingError, setFetchingError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'name', desc: false }
  ]);
  const { setPageTitle } = useAdminDashboard();

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
  
  const fetchFiles = async () => {
    setIsLoading(true);
    try {
      const { files, folders } = await listFiles(currentPath);
      setFiles(files);
      setFolders(folders);
      setFetchingError(null);
    } catch (error: any) {
      const errorMessage = error?.message ?? 'An unknown error occurred';
      setFetchingError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchFiles();
  }, [currentPath]); 

  // Combine files and folders into a single array for the table
  const combinedData: UploadItem[] = useMemo(() => {
    const folderItems: UploadItem[] = folders.map(folder => ({
      ...folder,
      type: 'folder' as const,
    }));
    
    const fileItems: UploadItem[] = files.map(file => ({
      ...file,
      type: 'file' as const,
    }));

    // Filter based on search query
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

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      try {
        await uploadFile(`${currentPath}${file.name}`, file);
        fetchFiles();
      } catch (error) {
        console.error('Error uploading file:', error);
      }
      // Reset the input
      event.target.value = '';
    }
  };

  const handleDeleteFile = async (key: string) => {
    try {
      await deleteFile(key);
      fetchFiles();
    } catch (error) {
      console.error('Error deleting file:', error);
    }
  };

  const handleCreateFolder = async (folderName: string) => {
    if (folderName) {
      try {
        await createFolder(`${currentPath}${folderName}/`);
        fetchFiles();
      } catch (error) {
        console.error('Error creating folder:', error);
      }
    }
  };

  const navigateToFolder = (path: string) => {
    setCurrentPath(path);
  };

  const navigateUp = () => {
    const newPath = currentPath.split('/').slice(0, -2).join('/') + '/';
    setCurrentPath(newPath === '/' ? '' : newPath);
  };

  const columns = useMemo(
    () => createColumns({
      onDelete: handleDeleteFile,
      onFolderClick: navigateToFolder,
    }),
    []
  );

  return (
    <div className="flex flex-col flex-1 p-0 md:p-4 mb-4 h-full overflow-hidden">
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
            onNavigateUp={navigateUp}
            onFileUpload={handleFileUpload}
            onCreateFolder={handleCreateFolder}
          />
    </div>
  );
}
