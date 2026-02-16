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
import { ScrollArea } from "@/components/ui/scroll-area";
import { useToast } from "@/hooks/use-toast";

export type UploadItem =
  | (FileData & { type: "file" })
  | (FolderData & { type: "folder" });

interface FileGridProps {
  data: UploadItem[];
  onDelete: (key: string) => void;
  onFolderClick: (path: string) => void;
  isLoading?: boolean;
}

const formatMarkdownLink = (file: FileData) => {
  return file.is_image
    ? `![${file.key}](${file.url})`
    : `[${file.key}](${file.url})`;
};

const formatShortDate = (d: Date | string) => {
  const date = typeof d === "string" ? new Date(d) : d;
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  if (diff < 86400000) return "Today";
  if (diff < 604800000)
    return `${Math.floor(diff / 86400000)}d ago`;
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
};

function FileDetailDialog({ file, onDelete }: { file: FileData; onDelete: (key: string) => void }) {
  const fileName = file.key.split("/").pop() || file.key;
  const { toast } = useToast();

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() =>
        toast({ title: "Copied", description: `${label} copied to clipboard` })
      )
      .catch(() =>
        toast({ title: "Failed to copy", variant: "destructive" })
      );
  };

  return (
    <DialogContent className="w-[90vw] h-[90vh] max-w-[90vw] sm:max-w-[90vw] flex flex-col p-0 gap-0 overflow-hidden !top-[5vh] !left-[5vw] !translate-x-0 !translate-y-0">
      <div className="flex flex-1 min-h-0 flex-col md:flex-row">
        {/* Left: Preview - 90% usable area (5% padding all sides) */}
        <div className="flex-1 min-w-0 flex items-center justify-center bg-muted/20 p-[5%]">
          {file.is_image ? (
            <img
              src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
              alt={fileName}
              className="max-h-full max-w-full w-auto object-contain"
            />
          ) : (
            <File className="w-24 h-24 text-muted-foreground" />
          )}
        </div>
        {/* Right: Metadata - scrollable */}
        <div className="w-full md:w-80 md:min-w-[20rem] border-t md:border-t-0 md:border-l flex flex-col min-h-0">
          <ScrollArea className="flex-1">
            <div className="p-4 space-y-4">
              <DialogHeader>
                <DialogTitle className="text-lg font-medium">
                  File Detail
                </DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <p className="text-sm font-medium">File name</p>
                  <p className="mt-1 break-all text-sm">{file.key}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Link</p>
                  <div className="flex mt-1 gap-2">
                    <a
                      href={file.url}
                      className="flex-1 min-w-0 break-all text-sm text-blue-600 dark:text-blue-400 hover:underline"
                    >
                      {file.url}
                    </a>
                    <Button
                      size="sm"
                      variant="outline"
                      className="shrink-0"
                      onClick={() => handleCopy(file.url, "Link")}
                    >
                      Copy
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium">Markdown</p>
                  <div className="flex mt-1 gap-2">
                    <p className="flex-1 min-w-0 p-2 bg-muted rounded text-sm font-mono break-all">
                      {formatMarkdownLink(file)}
                    </p>
                    <Button
                      size="sm"
                      variant="outline"
                      className="shrink-0"
                      onClick={() =>
                        handleCopy(formatMarkdownLink(file), "Markdown")
                      }
                    >
                      Copy
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium">Size</p>
                  <p className="mt-1 text-sm">{file.size}</p>
                </div>
                <div>
                  <p className="text-sm font-medium">Last modified</p>
                  <p className="mt-1 text-sm">
                    {file.last_modified
                      ? new Date(file.last_modified).toLocaleString()
                      : "—"}
                  </p>
                </div>
              </div>
            </div>
          </ScrollArea>
          <div className="p-4 border-t flex gap-2 shrink-0">
            <Button
              onClick={() => onDelete(file.key)}
              variant="destructive"
              size="sm"
            >
              Delete
            </Button>
            <DialogClose asChild>
              <Button type="button" variant="outline" size="sm">
                Close
              </Button>
            </DialogClose>
          </div>
        </div>
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
  const { toast } = useToast();

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() =>
        toast({ title: "Copied", description: `${label} copied to clipboard` })
      )
      .catch(() =>
        toast({ title: "Failed to copy", variant: "destructive" })
      );
  };

  return (
    <Dialog>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div className="group relative flex flex-col rounded-lg border bg-card overflow-hidden hover:bg-accent/50 transition-colors cursor-pointer">
            <DialogTrigger asChild>
              <button className="flex flex-col w-full text-left">
                {/* Upper: Preview (~70-75%) */}
                <div className="relative aspect-[4/3] flex items-center justify-center overflow-hidden bg-muted/30">
                  {file.is_image ? (
                    <img
                      src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
                      alt={fileName}
                      className="h-full w-full object-cover"
                      loading="lazy"
                    />
                  ) : (
                    <File className="h-12 w-12 text-muted-foreground" />
                  )}
                </div>
                {/* Lower: Metadata (~25-30%) */}
                <div className="px-2 py-1.5 flex flex-col gap-0.5">
                  <span className="text-xs truncate">{fileName}</span>
                  <div className="flex items-center justify-between gap-1 text-[10px] text-muted-foreground">
                    <span>{file.size}</span>
                    {file.last_modified && (
                      <span>{formatShortDate(file.last_modified)}</span>
                    )}
                  </div>
                </div>
              </button>
            </DialogTrigger>

            {/* Kebab menu overlaid on preview (top-right) */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute top-1 right-1 h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity bg-background/80 hover:bg-background"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleCopy(file.url, "URL")}>
                  <Copy className="mr-2 h-4 w-4" />
                  Copy URL
                </DropdownMenuItem>
                {file.is_image && (
                  <DropdownMenuItem
                    onClick={() =>
                      handleCopy(formatMarkdownLink(file), "Markdown")
                    }
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
          <ContextMenuItem onClick={() => handleCopy(file.url, "URL")}>
            <Copy className="mr-2 h-4 w-4" />
            Copy URL
          </ContextMenuItem>
          {file.is_image && (
            <ContextMenuItem
              onClick={() =>
                handleCopy(formatMarkdownLink(file), "Markdown")
              }
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
