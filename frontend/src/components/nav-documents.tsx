import React from "react"
import {
  IconDots,
  IconFolder,
  IconShare3,
  IconTrash,
  type Icon,
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
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { Link, useLocation, useNavigate } from "@tanstack/react-router"
import { Badge } from "@/components/ui/badge"
import { ArticleListItem } from "@/services/types"
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

  return (
    <>
      <SidebarGroup className="group-data-[collapsible=icon]:hidden">
        <div className="flex flex-row justify-between gap-2">
        <SidebarGroupLabel>Recent Articles</SidebarGroupLabel>
        </div>
        <SidebarMenu className="h-[calc(100vh-450px)] overflow-y-auto">
          {articles
            .slice()
            .sort((a, b) => {
              const dateA = a.article.created_at ? new Date(a.article.created_at).getTime() : 0;
              const dateB = b.article.created_at ? new Date(b.article.created_at).getTime() : 0;
              return dateB - dateA;
            })
            .map((articleItem) => {
              const editUrl = `/dashboard/blog/edit/${articleItem.article.slug || ''}`
              return (
                <SidebarMenuItem key={articleItem.article.id}>
                  <SidebarMenuButton isActive={location.pathname === editUrl} asChild>
                    <Link 
                      to={editUrl} 
                      className="flex flex-col items-start gap-1 p-2"
                      onClick={() => {
                        queryClient.invalidateQueries({ queryKey: ['article', articleItem.article.slug] })
                      }}
                    >
                      <div className="flex items-center gap-2 w-full">
                        <span className="text-sm font-medium truncate flex-1">{articleItem.article.title}</span>
                        <Badge 
                          className={`text-[0.6rem] ${
                            articleItem.article.is_draft 
                              ? "bg-indigo-50 dark:bg-indigo-900 text-indigo-700 dark:text-indigo-300" 
                              : "bg-green-50 dark:bg-green-900 text-green-700 dark:text-green-300"
                          }`} 
                          variant="outline"
                        >
                          {articleItem.article.is_draft ? 'Draft' : 'Published'}
                        </Badge>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {new Date(articleItem.article.created_at).toLocaleDateString()}
                      </span>
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
                              Are you sure you want to delete the article <b>{articleItem.article.title}</b>? This action cannot be undone.
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
