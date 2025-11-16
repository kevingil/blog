import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import { ArrowUpDown, MoreHorizontal, Folder, File, Trash2, Copy } from "lucide-react";
import { FileData, FolderData } from "@/services/storage";
import { VITE_PUBLIC_S3_URL_PREFIX } from "@/services/constants";

export type UploadItem = 
  | (FileData & { type: 'file' })
  | (FolderData & { type: 'folder' });

interface ColumnActions {
  onDelete: (key: string) => void;
  onFolderClick: (path: string) => void;
}

const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text).catch(err => {
    console.error('Error copying to clipboard:', err);
  });
};

const formatMarkdownLink = (file: FileData) => {
  const markdownLink = file.is_image
    ? `![${file.key}](${file.url})`
    : `[${file.key}](${file.url})`;
  return markdownLink;
};

export const createColumns = (actions: ColumnActions): ColumnDef<UploadItem>[] => [
  {
    accessorKey: "preview",
    header: "Type",
    cell: ({ row }) => {
      const item = row.original;
      if (item.type === 'folder') {
        return <Folder className="h-5 w-5 text-muted-foreground" />;
      }
      
      if (item.is_image) {
        return (
          <img
            src={`${VITE_PUBLIC_S3_URL_PREFIX}/${item.key}`}
            className="w-10 h-10 rounded object-cover"
            alt={item.key}
          />
        );
      }
      
      return <File className="h-5 w-5 text-muted-foreground" />;
    },
    enableSorting: false,
  },
  {
    accessorKey: "name",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Name
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const item = row.original;
      
      if (item.type === 'folder') {
        return (
          <Button 
            variant="link" 
            onClick={() => actions.onFolderClick(item.path)}
            className="p-0 h-auto font-normal"
          >
            {item.name}
          </Button>
        );
      }
      
      const fileName = item.key.split('/').pop() || item.key;
      return (
        <Dialog>
          <DialogTrigger className="text-left hover:underline">
            {fileName}
          </DialogTrigger>
          <DialogContent className="w-full md:max-w-5xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="text-xl font-medium">
                File Detail
              </DialogTitle>
            </DialogHeader>

            <div className="space-y-4">
              {item.is_image ? (
                <div className="flex justify-center">
                  <img
                    src={`${VITE_PUBLIC_S3_URL_PREFIX}/${item.key}`}
                    alt={item.key}
                    className="max-h-[500px] p-4"
                  />
                </div>
              ) : (
                <File className="w-full h-48 p-4 text-gray-600" />
              )}

              <div className="space-y-2">
                <div>
                  <p className="text-sm font-medium">File name</p>
                  <p className="mt-1">{item.key}</p>
                </div>

                <div>
                  <p className="text-sm font-medium">Link</p>
                  <div className="flex mt-1 gap-2">
                    <a href={item.url} className="mt-1 break-all text-blue-600 hover:underline">
                      {item.url}
                    </a>
                    <Button
                      className="px-3 rounded-l-none"
                      onClick={() => copyToClipboard(item.url)}
                    >
                      Copy
                    </Button>
                  </div>
                </div>

                <div>
                  <p className="text-sm font-medium">Markdown</p>
                  <div className="flex mt-1 gap-2">
                    <p className="flex-1 p-2 bg-gray-200 dark:bg-gray-800">
                      {formatMarkdownLink(item)}
                    </p>
                    <Button
                      className="px-3 rounded-l-none"
                      onClick={() => copyToClipboard(formatMarkdownLink(item))}
                    >
                      Copy
                    </Button>
                  </div>
                </div>

                <div>
                  <p className="text-sm font-medium">Size</p>
                  <p className="mt-1">{item.size}</p>
                </div>

                <div>
                  <p className="text-sm font-medium">Last modified</p>
                  <p className="mt-1">
                    {item.last_modified && new Date(item.last_modified).toLocaleString()}
                  </p>
                </div>
              </div>
            </div>

            <div className="mt-4 flex justify-between w-full gap-2">
              <Button
                onClick={() => actions.onDelete(item.key)}
                variant="destructive"
                className="w-full sm:w-auto"
              >
                Delete
              </Button>
              <DialogClose asChild>
                <Button type="button">
                  Close
                </Button>
              </DialogClose>
            </div>
          </DialogContent>
        </Dialog>
      );
    },
  },
  {
    accessorKey: "size",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Size
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const item = row.original;
      if (item.type === 'folder') {
        return <span className="text-muted-foreground">—</span>;
      }
      return <div className="text-sm">{item.size}</div>;
    },
  },
  {
    accessorKey: "last_modified",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Last Modified
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const item = row.original;
      const date = item.type === 'folder' ? item.lastModified : item.last_modified;
      
      if (!date) return <span className="text-muted-foreground">—</span>;
      
      return (
        <div className="text-sm">
          {new Date(date).toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </div>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const item = row.original;

      if (item.type === 'folder') {
        return (
          <Button 
            variant="ghost" 
            className="h-8 w-8 p-0"
            onClick={() => actions.onFolderClick(item.path)}
          >
            <span className="sr-only">Open folder</span>
            <Folder className="h-4 w-4" />
          </Button>
        );
      }

      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => copyToClipboard(item.url)}>
              <Copy className="mr-2 h-4 w-4" />
              Copy URL
            </DropdownMenuItem>
            {item.is_image && (
              <DropdownMenuItem onClick={() => copyToClipboard(formatMarkdownLink(item))}>
                <Copy className="mr-2 h-4 w-4" />
                Copy Markdown
              </DropdownMenuItem>
            )}
            <DropdownMenuItem 
              onClick={() => actions.onDelete(item.key)}
              className="text-destructive"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
];

