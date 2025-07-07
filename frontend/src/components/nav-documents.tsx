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
import { Link, useLocation } from "@tanstack/react-router"
import { Badge } from "@/components/ui/badge"
import { ArticleListItem } from "@/services/types"
import { useEffect, useRef } from "react"
import { FetchNextPageOptions, InfiniteQueryObserverResult, InfiniteData } from "@tanstack/react-query"
import { GetArticlesResponse } from "@/routes/dashboard/blog/index"

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
  const loadMoreRef = useRef<HTMLDivElement>(null)

  // Sort articles by created date, most recent first
  const sortedArticles = articles
    .sort((a, b) => new Date(b.article.created_at || 0).getTime() - new Date(a.article.created_at || 0).getTime())

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
    <SidebarGroup className="group-data-[collapsible=icon]:hidden">
      <div className="flex flex-row justify-between gap-2">
      <SidebarGroupLabel>Recent Articles</SidebarGroupLabel>
      </div>
      <SidebarMenu className="h-[calc(100vh-400px)] overflow-y-auto">
        {sortedArticles.map((articleItem) => {
          const editUrl = `/dashboard/blog/edit/${articleItem.article.slug || ''}`
          return (
            <SidebarMenuItem key={articleItem.article.id}>
              <SidebarMenuButton isActive={location.pathname === editUrl} asChild>
                <Link to={editUrl} className="flex flex-col items-start gap-1 p-2">
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
                  <DropdownMenuItem variant="destructive">
                    <IconTrash />
                    <span>Delete</span>
                  </DropdownMenuItem>
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
  )
}
