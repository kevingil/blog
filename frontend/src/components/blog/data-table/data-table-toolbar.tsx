import { Table } from "@tanstack/react-table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { X, Filter, Plus, Sparkles } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { GenerateArticleDrawer } from "@/components/blog/GenerateArticleDrawer";

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  statusFilter: "all" | "published" | "drafts";
  onStatusFilterChange: (value: "all" | "published" | "drafts") => void;
}

export function DataTableToolbar<TData>({
  table,
  searchQuery,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
}: DataTableToolbarProps<TData>) {
  const isFiltered = searchQuery.length > 0 || statusFilter !== "published";

  return (
    <div className="flex items-center justify-between py-4">
      <div className="flex flex-1 items-center space-x-2">
        <Input
          placeholder="Search articles..."
          value={searchQuery}
          onChange={(event) => onSearchChange(event.target.value)}
          className="h-9 w-[150px] lg:w-[250px]"
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-9 border-dashed">
              <Filter className="mr-2 h-4 w-4" />
              Status
              {statusFilter !== "published" && (
                <span className="ml-2 rounded-sm bg-primary px-1 text-[0.6rem] font-semibold text-primary-foreground">
                  {statusFilter === "all"
                    ? "All"
                    : statusFilter === "drafts"
                    ? "Drafts"
                    : "Published"}
                </span>
              )}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-[150px]">
            <DropdownMenuRadioGroup
              value={statusFilter}
              onValueChange={(value) =>
                onStatusFilterChange(value as "all" | "published" | "drafts")
              }
            >
              <DropdownMenuRadioItem value="all">All</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="published">
                Published
              </DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="drafts">
                Drafts
              </DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => {
              onSearchChange("");
              onStatusFilterChange("published");
            }}
            className="h-9 px-2 lg:px-3"
          >
            Reset
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="flex items-center gap-4">
        <GenerateArticleDrawer>
          <Button>
            <Sparkles className="mr-2 h-4 w-4" />
            Generate
          </Button>
        </GenerateArticleDrawer>
        <Link to="/dashboard/blog/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            New Article
          </Button>
        </Link>
      </div>
    </div>
  );
}

