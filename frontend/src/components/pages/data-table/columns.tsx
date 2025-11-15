import { ColumnDef } from "@tanstack/react-table";
import { Page } from "@/services/pages";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Link } from "@tanstack/react-router";
import { ArrowUpDown, MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import { deletePage } from "@/services/pages";

export const createColumns = (onPageDeleted: () => void): ColumnDef<Page>[] => [
  {
    accessorKey: "title",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Title
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const page = row.original;
      return (
        <div className="flex flex-col gap-1 py-2">
          <Link
            to="/dashboard/pages/edit/$pageId"
            params={{ pageId: page.id }}
            className="font-medium hover:underline"
          >
            {page.title}
          </Link>
          {page.description && (
            <p className="text-xs text-muted-foreground line-clamp-1">
              {page.description}
            </p>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: "slug",
    header: "Slug",
    cell: ({ row }) => {
      return (
        <div className="text-sm font-mono text-muted-foreground">
          /{row.original.slug}
        </div>
      );
    },
  },
  {
    accessorKey: "is_published",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Status
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const isPublished = row.original.is_published;
      return (
        <Badge
          variant="outline"
          className={`text-[0.65rem] ${
            isPublished
              ? "bg-green-50 dark:bg-green-900/30"
              : "bg-gray-50 dark:bg-gray-900/30"
          }`}
        >
          {isPublished ? "Published" : "Draft"}
        </Badge>
      );
    },
  },
  {
    accessorKey: "updated_at",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Updated
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const date = new Date(row.original.updated_at);
      return (
        <div className="text-sm">
          {date.toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
          })}
        </div>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const page = row.original;

      const handleDelete = async () => {
        if (!confirm(`Are you sure you want to delete "${page.title}"?`)) {
          return;
        }

        try {
          await deletePage(page.id);
          onPageDeleted();
        } catch (error) {
          console.error("Failed to delete page:", error);
          alert("Failed to delete page");
        }
      };

      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link
                to="/dashboard/pages/edit/$pageId"
                params={{ pageId: page.id }}
              >
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleDelete}>
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

