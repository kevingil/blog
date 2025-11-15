import { ColumnDef } from "@tanstack/react-table";
import { ArticleListItem } from "@/services/types";
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
import { deleteArticle } from "@/services/blog";

export const createColumns = (onArticleDeleted: () => void): ColumnDef<ArticleListItem>[] => [
  {
    accessorKey: "article.title",
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
      const article = row.original.article;
      return (
        <div className="flex items-start gap-3 py-2">
          {article.image_url && (
            <img
              src={article.image_url}
              className="rounded-md w-16 h-16 min-w-16 min-h-16 object-cover"
              alt={article.title}
            />
          )}
          <div className="flex flex-col gap-1 min-w-0">
            <Link
              to="/dashboard/blog/edit/$blogSlug"
              params={{ blogSlug: article.slug || "" }}
              className="font-medium hover:underline truncate"
            >
              {article.title}
            </Link>
            {article.content && (
              <p className="text-xs text-muted-foreground line-clamp-2">
                {article.content.slice(0, 150)}...
              </p>
            )}
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "tags",
    header: "Tags",
    cell: ({ row }) => {
      const tags = row.original.tags;
      if (!tags || tags.length === 0) return null;
      return (
        <div className="flex flex-wrap gap-1">
          {tags
            .filter((tag) => tag.name !== null && tag.name !== "")
            .slice(0, 3)
            .map((tag) => (
              <Badge
                key={tag.tag_id}
                variant="outline"
                className="text-[0.65rem] px-1.5 py-0"
              >
                {tag.name.toUpperCase()}
              </Badge>
            ))}
          {tags.length > 3 && (
            <Badge variant="outline" className="text-[0.65rem] px-1.5 py-0">
              +{tags.length - 3}
            </Badge>
          )}
        </div>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: "article.created_at",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Created
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const date = new Date(row.original.article.created_at);
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
    accessorKey: "article.published_at",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          className="h-8 px-2"
        >
          Published
          <ArrowUpDown className="ml-2 h-4 w-4" />
        </Button>
      );
    },
    cell: ({ row }) => {
      const publishedAt = row.original.article.published_at;
      if (!publishedAt) return <span className="text-sm text-muted-foreground">—</span>;
      const date = new Date(publishedAt);
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
    accessorKey: "article.is_draft",
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
      const isDraft = row.original.article.is_draft;
      return (
        <Badge
          variant="outline"
          className={`text-[0.65rem] ${
            isDraft
              ? "bg-indigo-50 dark:bg-indigo-900/30"
              : "bg-green-50 dark:bg-green-900/30"
          }`}
        >
          {isDraft ? "Draft" : "Published"}
        </Badge>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const article = row.original.article;

      const handleDelete = async () => {
        const result = await deleteArticle(article.id);
        if (result.success) {
          onArticleDeleted();
        } else {
          console.error("Failed to delete article");
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
                to="/dashboard/blog/edit/$blogSlug"
                params={{ blogSlug: article.slug || "" }}
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

