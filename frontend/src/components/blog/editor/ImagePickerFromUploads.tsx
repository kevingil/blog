import { useState } from 'react';
import { Folder, ChevronLeft } from 'lucide-react';
import { useStorageFiles } from '@/hooks/use-storage-files';
import { cn } from '@/lib/utils';

interface ImagePickerFromUploadsProps {
  onSelect: (url: string) => void;
}

function getParentPath(currentPath: string): string {
  const trimmed = currentPath.replace(/\/$/, '');
  if (!trimmed) return '';
  const parts = trimmed.split('/').filter(Boolean);
  if (parts.length <= 1) return '';
  return parts.slice(0, -1).join('/') + '/';
}

export function ImagePickerFromUploads({ onSelect }: ImagePickerFromUploadsProps) {
  const [currentPath, setCurrentPath] = useState('');

  const { data, isLoading, error } = useStorageFiles(currentPath);

  const files = data?.files ?? [];
  const folders = data?.folders ?? [];
  const imageFiles = files.filter((f) => f.is_image);
  const parentPath = getParentPath(currentPath);

  if (error) {
    return (
      <div className="flex items-center justify-center h-48 text-destructive text-sm">
        {error instanceof Error ? error.message : 'Failed to load uploads'}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground text-sm">
        Loading...
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Breadcrumb / navigation */}
      <div className="flex items-center gap-2">
        {currentPath ? (
          <button
            type="button"
            onClick={() => setCurrentPath(parentPath)}
            className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
          >
            <ChevronLeft className="w-4 h-4" />
            Back
          </button>
        ) : null}
        <span className="text-xs text-muted-foreground truncate flex-1">
          {currentPath || '/'}
        </span>
      </div>

      {/* Folders and images grid */}
      <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-2 max-h-64 overflow-y-auto">
        {folders.map((folder) => (
          <button
            key={folder.path}
            type="button"
            onClick={() => setCurrentPath(folder.path)}
            className="flex flex-col items-center gap-2 p-2 rounded-lg border bg-card hover:bg-accent/50 transition-colors text-center"
          >
            <Folder className="h-10 w-10 text-blue-500" />
            <span className="text-xs truncate w-full">{folder.name}</span>
          </button>
        ))}
        {imageFiles.map((file) => {
          const fileName = file.key.split('/').pop() || file.key;
          return (
            <button
              key={file.key}
              type="button"
              onClick={() => onSelect(file.url)}
              className={cn(
                'flex flex-col rounded-lg border overflow-hidden hover:ring-2 hover:ring-primary transition-all',
                'bg-card hover:bg-accent/50'
              )}
            >
              <div className="aspect-square flex items-center justify-center overflow-hidden bg-muted/30">
                <img
                  src={file.url}
                  alt={fileName}
                  className="h-full w-full object-cover"
                  loading="lazy"
                />
              </div>
              <span className="text-[10px] truncate px-1 py-0.5 text-muted-foreground">
                {fileName}
              </span>
            </button>
          );
        })}
      </div>

      {folders.length === 0 && imageFiles.length === 0 && (
        <div className="flex flex-col items-center justify-center h-24 text-muted-foreground text-sm">
          No images in this folder
        </div>
      )}
    </div>
  );
}
