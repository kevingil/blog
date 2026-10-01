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
    <div className="flex w-full min-w-0 flex-col gap-2 py-3 md:flex-row md:items-center md:justify-between md:py-4">
      <div className="flex min-w-0 items-center gap-2">
        <Input
          placeholder="Search articles..."
          value={searchQuery}
          onChange={(event) => onSearchChange(event.target.value)}
          className="h-9 min-w-0 flex-1 md:w-[150px] md:flex-none lg:w-[250px]"
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-9 shrink-0 border-dashed">
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
            className="h-9 shrink-0 px-2 lg:px-3"
          >
            Reset
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="flex min-w-0 items-center gap-2">
        <Link to="/dashboard" className="min-w-0 flex-1 md:flex-none">
          <Button size="sm" className="h-9 w-full">
            <Sparkles className="h-4 w-4" />
            Generate
          </Button>
        </Link>
        <Link to="/dashboard/blog/new" className="min-w-0 flex-1 md:flex-none">
          <Button size="sm" variant="outline" className="h-9 w-full">
            <Plus className="h-4 w-4" />
            <span className="md:hidden">New</span>
            <span className="hidden md:inline">New Article</span>
          </Button>
        </Link>
      </div>
    </div>
  );
}

