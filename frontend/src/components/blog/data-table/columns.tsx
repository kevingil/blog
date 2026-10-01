import { ColumnDef } from "@tanstack/react-table";
import { ArticleListItem, isPublished, hasDraftChanges, externalArticleUrl } from "@/services/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Link } from "@tanstack/react-router";
import { ArrowUpDown, ExternalLink, MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import { deleteArticle } from "@/services/blog";

export const createColumns = (onArticleDeleted: () => void): ColumnDef<ArticleListItem>[] => [
  {
    accessorKey: "article.draft_title",
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
      const hasChanges = hasDraftChanges(article);
      const externalUrl = externalArticleUrl(article);
      return (
        <div className="flex min-w-0 items-start gap-2 py-1 md:gap-3 md:py-2">
          {article.draft_image_url && (
            <img
              src={article.draft_image_url}
              className="size-10 shrink-0 rounded-md object-cover md:size-16"
              alt={article.draft_title}
            />
          )}
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex min-w-0 items-center gap-1.5">
              <Link
                to="/dashboard/blog/edit/$blogSlug"
                params={{ blogSlug: article.slug || "" }}
                className="min-w-0 truncate font-medium hover:underline"
              >
                {article.draft_title}
              </Link>
              {externalUrl && (
                <Badge variant="outline" className="shrink-0 px-1 py-0 text-[0.6rem]">
                  external
                </Badge>
              )}
              {hasChanges && (
                <Badge variant="outline" className="shrink-0 border-amber-300 px-1 py-0 text-[0.6rem] text-amber-600 dark:text-amber-400">
                  modified
                </Badge>
              )}
            </div>
            {article.draft_content && (
              <p className="truncate text-xs text-muted-foreground">
                {article.draft_content.replace(/<[^>]*>/g, '').slice(0, 150)}
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
    accessorKey: "article.published_at",
    id: "status",
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
      const article = row.original.article;
      const published = isPublished(article);
      const external = Boolean(externalArticleUrl(article));
      return (
        <Badge
          variant="outline"
          className={`text-[0.65rem] ${
            published
              ? "bg-green-50 dark:bg-green-900/30"
              : "bg-indigo-50 dark:bg-indigo-900/30"
          }`}
        >
          {external ? "External" : published ? "Published" : "Draft"}
        </Badge>
      );
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const article = row.original.article;
      const externalUrl = externalArticleUrl(article);

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
            {externalUrl && (
              <DropdownMenuItem asChild>
                <a href={externalUrl} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Open link
                </a>
              </DropdownMenuItem>
            )}
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

