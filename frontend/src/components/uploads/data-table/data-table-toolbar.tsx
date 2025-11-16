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
import { X, Upload, FolderPlus, ChevronUp } from "lucide-react";
import { useState } from "react";

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  currentPath: string;
  onNavigateUp: () => void;
  onFileUpload: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onCreateFolder: (name: string) => void;
}

export function DataTableToolbar<TData>({
  table,
  searchQuery,
  onSearchChange,
  currentPath,
  onNavigateUp,
  onFileUpload,
  onCreateFolder,
}: DataTableToolbarProps<TData>) {
  const [newFolderName, setNewFolderName] = useState('');
  const [isCreateFolderOpen, setIsCreateFolderOpen] = useState(false);
  const isFiltered = searchQuery.length > 0;

  const handleCreateFolder = () => {
    if (newFolderName.trim()) {
      onCreateFolder(newFolderName.trim());
      setNewFolderName('');
      setIsCreateFolderOpen(false);
    }
  };

  return (
    <div className="space-y-4 py-4">
      {/* Breadcrumb Navigation */}
      <div className="flex items-center gap-2">
        <Button 
          onClick={onNavigateUp} 
          disabled={currentPath === ''}
          variant="outline"
          size="sm"
          className="h-9"
        >
          <ChevronUp className="mr-2 h-4 w-4" />
          Up
        </Button>
        <span className="text-sm text-muted-foreground">
          {currentPath || '/'}
        </span>
      </div>

      {/* Search and Actions */}
      <div className="flex items-center justify-between">
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
        <div className="flex items-center gap-2">
          {/* Upload File */}
          <div>
            <Input
              id="file-upload"
              type="file"
              onChange={onFileUpload}
              className="hidden"
            />
            <Button asChild variant="outline" size="sm" className="h-9">
              <label htmlFor="file-upload" className="cursor-pointer">
                <Upload className="mr-2 h-4 w-4" />
                Upload File
              </label>
            </Button>
          </div>

          {/* Create Folder Dialog */}
          <Dialog open={isCreateFolderOpen} onOpenChange={setIsCreateFolderOpen}>
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
                    if (e.key === 'Enter') {
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

