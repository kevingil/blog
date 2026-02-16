import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Folder, File, Trash2, Copy, MoreVertical, FolderOpen, Image } from "lucide-react";
import { FileData, FolderData } from "@/services/storage";
import { VITE_PUBLIC_S3_URL_PREFIX } from "@/services/constants";

export type UploadItem =
  | (FileData & { type: "file" })
  | (FolderData & { type: "folder" });

interface FileGridProps {
  data: UploadItem[];
  onDelete: (key: string) => void;
  onFolderClick: (path: string) => void;
  isLoading?: boolean;
}

const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text).catch((err) => {
    console.error("[FileGrid] clipboard write failed:", err);
  });
};

const formatMarkdownLink = (file: FileData) => {
  return file.is_image
    ? `![${file.key}](${file.url})`
    : `[${file.key}](${file.url})`;
};

function FileDetailDialog({ file, onDelete }: { file: FileData; onDelete: (key: string) => void }) {
  const fileName = file.key.split("/").pop() || file.key;

  return (
    <DialogContent className="w-full md:max-w-5xl max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle className="text-xl font-medium">File Detail</DialogTitle>
      </DialogHeader>
      <div className="space-y-4">
        {file.is_image ? (
          <div className="flex justify-center">
            <img
              src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
              alt={fileName}
              className="max-h-[500px] p-4"
            />
          </div>
        ) : (
          <div className="flex justify-center py-8">
            <File className="w-24 h-24 text-muted-foreground" />
          </div>
        )}
        <div className="space-y-2">
          <div>
            <p className="text-sm font-medium">File name</p>
            <p className="mt-1">{file.key}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Link</p>
            <div className="flex mt-1 gap-2">
              <a
                href={file.url}
                className="mt-1 break-all text-blue-600 hover:underline"
              >
                {file.url}
              </a>
              <Button
                className="px-3 rounded-l-none"
                onClick={() => copyToClipboard(file.url)}
              >
                Copy
              </Button>
            </div>
          </div>
          <div>
            <p className="text-sm font-medium">Markdown</p>
            <div className="flex mt-1 gap-2">
              <p className="flex-1 p-2 bg-gray-200 dark:bg-gray-800 rounded text-sm font-mono break-all">
                {formatMarkdownLink(file)}
              </p>
              <Button
                className="px-3 rounded-l-none"
                onClick={() => copyToClipboard(formatMarkdownLink(file))}
              >
                Copy
              </Button>
            </div>
          </div>
          <div>
            <p className="text-sm font-medium">Size</p>
            <p className="mt-1">{file.size}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Last modified</p>
            <p className="mt-1">
              {file.last_modified &&
                new Date(file.last_modified).toLocaleString()}
            </p>
          </div>
        </div>
      </div>
      <div className="mt-4 flex justify-between w-full gap-2">
        <Button
          onClick={() => onDelete(file.key)}
          variant="destructive"
          className="w-full sm:w-auto"
        >
          Delete
        </Button>
        <DialogClose asChild>
          <Button type="button">Close</Button>
        </DialogClose>
      </div>
    </DialogContent>
  );
}

function FolderCard({
  folder,
  onFolderClick,
}: {
  folder: FolderData & { type: "folder" };
  onFolderClick: (path: string) => void;
}) {
  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <button
          className="group flex flex-col items-center gap-2 p-4 rounded-lg border bg-card hover:bg-accent/50 transition-colors cursor-pointer text-center w-full"
          onDoubleClick={() => onFolderClick(folder.path)}
          onClick={() => onFolderClick(folder.path)}
        >
          <Folder className="h-12 w-12 text-blue-500 group-hover:text-blue-400 transition-colors" />
          <span className="text-sm font-medium truncate w-full">
            {folder.name}
          </span>
          {folder.fileCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {folder.fileCount} item{folder.fileCount !== 1 ? "s" : ""}
            </span>
          )}
        </button>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={() => onFolderClick(folder.path)}>
          <FolderOpen className="mr-2 h-4 w-4" />
          Open
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}

function FileCard({
  file,
  onDelete,
}: {
  file: FileData & { type: "file" };
  onDelete: (key: string) => void;
}) {
  const fileName = file.key.split("/").pop() || file.key;

  return (
    <Dialog>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div className="group relative flex flex-col items-center gap-2 p-4 rounded-lg border bg-card hover:bg-accent/50 transition-colors cursor-pointer text-center">
            <DialogTrigger asChild>
              <button className="flex flex-col items-center gap-2 w-full">
                {file.is_image ? (
                  <div className="h-20 w-full flex items-center justify-center overflow-hidden rounded">
                    <img
                      src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
                      alt={fileName}
                      className="max-h-20 max-w-full object-contain"
                      loading="lazy"
                    />
                  </div>
                ) : (
                  <File className="h-12 w-12 text-muted-foreground" />
                )}
                <span className="text-sm truncate w-full">{fileName}</span>
                <span className="text-xs text-muted-foreground">
                  {file.size}
                </span>
              </button>
            </DialogTrigger>

            {/* Action button (top-right) */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute top-1 right-1 h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => copyToClipboard(file.url)}>
                  <Copy className="mr-2 h-4 w-4" />
                  Copy URL
                </DropdownMenuItem>
                {file.is_image && (
                  <DropdownMenuItem
                    onClick={() => copyToClipboard(formatMarkdownLink(file))}
                  >
                    <Image className="mr-2 h-4 w-4" />
                    Copy Markdown
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={() => onDelete(file.key)}
                  className="text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </ContextMenuTrigger>

        <ContextMenuContent>
          <ContextMenuItem onClick={() => copyToClipboard(file.url)}>
            <Copy className="mr-2 h-4 w-4" />
            Copy URL
          </ContextMenuItem>
          {file.is_image && (
            <ContextMenuItem
              onClick={() => copyToClipboard(formatMarkdownLink(file))}
            >
              <Image className="mr-2 h-4 w-4" />
              Copy Markdown
            </ContextMenuItem>
          )}
          <ContextMenuSeparator />
          <ContextMenuItem
            onClick={() => onDelete(file.key)}
            variant="destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>

      <FileDetailDialog file={file} onDelete={onDelete} />
    </Dialog>
  );
}

export function FileGrid({
  data,
  onDelete,
  onFolderClick,
  isLoading = false,
}: FileGridProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        Loading...
      </div>
    );
  }

  if (data.length === 0) {
    return null;
  }

  const folders = data.filter(
    (item): item is FolderData & { type: "folder" } => item.type === "folder"
  );
  const files = data.filter(
    (item): item is FileData & { type: "file" } => item.type === "file"
  );

  return (
    <div className="space-y-6 pb-4">
      {folders.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            Folders
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {folders.map((folder) => (
              <FolderCard
                key={folder.path}
                folder={folder}
                onFolderClick={onFolderClick}
              />
            ))}
          </div>
        </div>
      )}
      {files.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            Files
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {files.map((file) => (
              <FileCard
                key={file.key}
                file={file}
                onDelete={onDelete}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
