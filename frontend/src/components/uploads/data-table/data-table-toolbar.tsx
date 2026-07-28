import { Table } from "@tanstack/react-table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { X, Upload, FolderPlus, LayoutGrid, List, Home } from "lucide-react";
import { useState, Fragment } from "react";

export type ViewMode = "grid" | "list";

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  currentPath: string;
  onNavigateToPath: (path: string) => void;
  onFileUpload: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onCreateFolder: (name: string) => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
}

function PathBreadcrumb({
  currentPath,
  onNavigateToPath,
}: {
  currentPath: string;
  onNavigateToPath: (path: string) => void;
}) {
  const segments = currentPath
    .split("/")
    .filter((s) => s.length > 0);

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          {segments.length === 0 ? (
            <BreadcrumbPage className="flex items-center gap-1.5">
              <Home className="h-3.5 w-3.5" />
              Files
            </BreadcrumbPage>
          ) : (
            <BreadcrumbLink
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onNavigateToPath("");
              }}
              className="flex items-center gap-1.5"
            >
              <Home className="h-3.5 w-3.5" />
              Files
            </BreadcrumbLink>
          )}
        </BreadcrumbItem>

        {segments.map((segment, index) => {
          const pathUpToHere =
            segments.slice(0, index + 1).join("/") + "/";
          const isLast = index === segments.length - 1;

          return (
            <Fragment key={pathUpToHere}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{segment}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      onNavigateToPath(pathUpToHere);
                    }}
                  >
                    {segment}
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}

export function DataTableToolbar<TData>({
  searchQuery,
  onSearchChange,
  currentPath,
  onNavigateToPath,
  onFileUpload,
  onCreateFolder,
  viewMode,
  onViewModeChange,
}: DataTableToolbarProps<TData>) {
  const [newFolderName, setNewFolderName] = useState("");
  const [isCreateFolderOpen, setIsCreateFolderOpen] = useState(false);
  const isFiltered = searchQuery.length > 0;

  const handleCreateFolder = () => {
    if (newFolderName.trim()) {
      onCreateFolder(newFolderName.trim());
      setNewFolderName("");
      setIsCreateFolderOpen(false);
    }
  };

  return (
    <div className="space-y-4 py-4">
      {/* Breadcrumb Navigation */}
      <PathBreadcrumb
        currentPath={currentPath}
        onNavigateToPath={onNavigateToPath}
      />

      {/* Search and Actions */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-1 items-center space-x-2">
          <Input
            placeholder="Search files and folders..."
            value={searchQuery}
            onChange={(event) => onSearchChange(event.target.value)}
            className="h-9 w-[150px] lg:w-[250px]"
          />
          {isFiltered && (
            <Button
              variant="ghost"
              onClick={() => onSearchChange("")}
              className="h-9 px-2 lg:px-3"
            >
              Reset
              <X className="ml-2 h-4 w-4" />
            </Button>
          )}
        </div>
        <div className="flex items-center gap-1">
          {/* View Toggle */}
          <div className="flex items-center border rounded-md">
            <Button
              variant={viewMode === "grid" ? "secondary" : "ghost"}
              size="icon"
              className="h-9 w-9 rounded-r-none"
              onClick={() => onViewModeChange("grid")}
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === "list" ? "secondary" : "ghost"}
              size="icon"
              className="h-9 w-9 rounded-l-none"
              onClick={() => onViewModeChange("list")}
            >
              <List className="h-4 w-4" />
            </Button>
          </div>

          {/* Upload File */}
          <div>
            <Input
              id="file-upload"
              type="file"
              multiple
              onChange={onFileUpload}
              className="hidden"
            />
            <Button asChild variant="outline" size="sm" className="h-9">
              <label htmlFor="file-upload" className="cursor-pointer">
                <Upload className="mr-2 h-4 w-4" />
                Upload
              </label>
            </Button>
          </div>

          {/* Create Folder Dialog */}
          <Dialog
            open={isCreateFolderOpen}
            onOpenChange={setIsCreateFolderOpen}
          >
            <DialogTrigger asChild>
              <Button variant="outline" size="sm" className="h-9">
                <FolderPlus className="mr-2 h-4 w-4" />
                New Folder
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Folder</DialogTitle>
              </DialogHeader>
              <div className="py-4">
                <Input
                  placeholder="Folder name"
                  value={newFolderName}
                  onChange={(e) => setNewFolderName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      handleCreateFolder();
                    }
                  }}
                />
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="outline">Cancel</Button>
                </DialogClose>
                <Button onClick={handleCreateFolder}>Create</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>
    </div>
  );
}
