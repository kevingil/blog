import React from "react"
import {
  IconDots,
  IconFolder,
  IconSearch,
  IconTrash,
} from "@tabler/icons-react"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarGroup,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { Link, useLocation, useNavigate } from "@tanstack/react-router"
import { ArticleListItem, isPublished } from "@/services/types"

function formatArticleMeta(article: ArticleListItem["article"]): string {
  const fmt = (s: string) =>
    new Date(s).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })
  if (isPublished(article)) {
    return `Published ${fmt(article.published_at!)}`
  }
  return article.created_at ? `Created ${fmt(article.created_at)} · Draft` : "Draft"
}
import { useEffect, useRef } from "react"
import { FetchNextPageOptions, InfiniteQueryObserverResult, InfiniteData, useQueryClient } from "@tanstack/react-query"
import { GetArticlesResponse } from "@/routes/dashboard/blog/index"
import { useMutation } from "@tanstack/react-query"
import { deleteArticle } from "@/services/blog"
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog"

interface NavDocumentsProps {
  articles: ArticleListItem[]
  fetchNextPage: (options?: FetchNextPageOptions) => Promise<InfiniteQueryObserverResult<InfiniteData<GetArticlesResponse, unknown>, Error>>
  hasNextPage: boolean
  isFetchingNextPage: boolean
}

export function NavDocuments({
  articles,
  fetchNextPage,
  hasNextPage,
  isFetchingNextPage,
}: NavDocumentsProps) {
  const { isMobile } = useSidebar()
  const location = useLocation()
  const navigate = useNavigate()
  const loadMoreRef = useRef<HTMLDivElement>(null)
  const queryClient = useQueryClient()
  const [articleToDelete, setArticleToDelete] = React.useState<ArticleListItem | null>(null)
  const [searchOpen, setSearchOpen] = React.useState(false)
  const [query, setQuery] = React.useState("")

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return await deleteArticle(id)
    },
    onSuccess: (_, id: string) => {
      queryClient.setQueryData(['sidebar-articles', 'all'], (oldData: any) => {
        if (!oldData) return oldData;
        return {
          ...oldData,
          pages: oldData.pages.map((page: any) => ({
            ...page,
            articles: page.articles.filter((a: any) => a.article.id !== id),
          })),
        };
      });
      if (
        articleToDelete &&
        articleToDelete.article.slug &&
        location.pathname.includes(articleToDelete.article.slug)
      ) {
        navigate({ to: "/dashboard/blog" });
      }
      // Do NOT call setArticleToDelete(null) here!
    },
  })

  // Intersection Observer for infinite scrolling
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        const first = entries[0]
        if (first.isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 1 }
    )

    if (loadMoreRef.current) {
      observer.observe(loadMoreRef.current)
    }

    return () => {
      if (loadMoreRef.current) {
        observer.unobserve(loadMoreRef.current)
      }
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage])

  const normalizedQuery = query.trim().toLowerCase()
  const visibleArticles = articles
    .slice()
    .sort((a, b) => {
      const dateA = a.article.created_at ? new Date(a.article.created_at).getTime() : 0;
      const dateB = b.article.created_at ? new Date(b.article.created_at).getTime() : 0;
      return dateB - dateA;
    })
    .filter((articleItem) => {
      if (!normalizedQuery) return true
      return (articleItem.article.draft_title || "").toLowerCase().includes(normalizedQuery)
    })

  return (
    <>
      <SidebarGroup className="group-data-[collapsible=icon]:hidden flex-1 flex flex-col min-h-0">
        <div className="flex items-center gap-2 px-2 pb-1">
          <span className="text-sm text-sidebar-foreground/55">Recent</span>
          <button
            type="button"
            aria-label="Search articles"
            aria-expanded={searchOpen}
            onClick={() => {
              setSearchOpen((open) => !open)
              setQuery("")
            }}
            className="ml-auto inline-flex size-7 items-center justify-center rounded-md text-sidebar-foreground/55 hover:bg-sidebar-accent hover:text-sidebar-foreground"
          >
            <IconSearch className="size-4" />
          </button>
        </div>
        {searchOpen && (
          <input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search articles"
            className="mx-2 mb-2 h-8 rounded-lg border border-sidebar-border bg-transparent px-2.5 text-sm outline-none focus:ring-2 focus:ring-sidebar-ring/40"
          />
        )}
        <SidebarMenu className="flex-1 overflow-y-auto min-h-0">
          {visibleArticles.map((articleItem) => {
              const editUrl = `/dashboard/blog/edit/${articleItem.article.slug || ''}`
              return (
                <SidebarMenuItem key={articleItem.article.id}>
                  <SidebarMenuButton isActive={location.pathname === editUrl} asChild className="h-8 rounded-lg font-normal">
                    <Link 
                      to={editUrl} 
                      title={formatArticleMeta(articleItem.article)}
                      className="px-2"
                      onClick={() => {
                        queryClient.invalidateQueries({ queryKey: ['article', articleItem.article.slug] })
                      }}
                    >
                      <span className="truncate text-sm text-sidebar-foreground/85">{articleItem.article.draft_title}</span>
                    </Link>
                  </SidebarMenuButton>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <SidebarMenuAction
                        showOnHover
                        className="data-[state=open]:bg-accent rounded-sm"
                      >
                        <IconDots />
                        <span className="sr-only">More</span>
                      </SidebarMenuAction>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      className="w-24 rounded-lg"
                      side={isMobile ? "bottom" : "right"}
                      align={isMobile ? "end" : "start"}
                    >
                      <DropdownMenuItem asChild>
                        <Link to={editUrl}>
                          <IconFolder />
                          <span>Edit</span>
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      {/* AlertDialog is now inside DropdownMenuContent for each article */}
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <DropdownMenuItem
                            variant="destructive"
                            onSelect={e => {
                              e.preventDefault();
                              setArticleToDelete(articleItem);
                            }}
                            disabled={deleteMutation.isPending}
                          >
                            <IconTrash />
                            <span>Delete</span>
                          </DropdownMenuItem>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Delete Article?</AlertDialogTitle>
                            <AlertDialogDescription>
                              Are you sure you want to delete the article <b>{articleItem.article.draft_title}</b>? This action cannot be undone.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel asChild>
                              <button type="button">Cancel</button>
                            </AlertDialogCancel>
                            <AlertDialogAction asChild>
                              <button
                                type="button"
                                onClick={() => {
                                  deleteMutation.mutate(articleItem.article.id);
                                }}
                                disabled={deleteMutation.isPending}
                              >
                                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
                              </button>
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </SidebarMenuItem>
              )
            })}
          {normalizedQuery && visibleArticles.length === 0 && (
            <p className="px-2 py-3 text-xs text-sidebar-foreground/50">No matches</p>
          )}
          {/* Loading indicator for infinite scroll */}
          {hasNextPage && (
            <div ref={loadMoreRef} className="flex justify-center p-4">
              {isFetchingNextPage ? (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
              ) : (
                <div className="text-xs text-muted-foreground">Scroll for more</div>
              )}
            </div>
          )}
        </SidebarMenu>
      </SidebarGroup>
    </>
  )
}
