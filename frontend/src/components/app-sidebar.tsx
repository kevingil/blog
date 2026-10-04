import * as React from "react"
import {
  IconBulb,
  IconCamera,
  IconFileDescription,
  IconFileWord,
  IconFolder,
  IconInnerShadowTop,
  IconBook,
  IconLink,
  IconPlug,
  IconUsers,
  IconUpload,
} from "@tabler/icons-react"

import { NavDocuments } from "@/components/nav-documents"
import { NavMain } from "@/components/nav-main"
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar"
import { useLocation, useNavigate } from "@tanstack/react-router"
import { useInfiniteQuery } from "@tanstack/react-query"
import { getArticles } from "@/services/blog"
import { Plus } from "lucide-react"


const navigationData = {
  navMain: [
    {
      title: "Articles",
      url: "/dashboard/blog",
      icon: IconFileDescription,
      items: [
        {
          title: "All Articles",
          url: "/dashboard/blog",
        },
        {
          title: "Draft Articles",
          url: "/dashboard/blog?status=draft",
        },
        {
          title: "Published Articles",
          url: "/dashboard/blog?status=published",
        },
        {
          title: "New Article",
          url: "/dashboard/blog/new",
        },
      ],
    },
    {
      title: "Pages",
      url: "/dashboard/pages",
      icon: IconFileWord,
      items: [
        {
          title: "All Pages",
          url: "/dashboard/pages",
        },
        {
          title: "New Page",
          url: "/dashboard/pages/new",
        },
      ],
    },
    {
      title: "Projects",
      url: "/dashboard/projects",
      icon: IconFolder,
      items: [
        {
          title: "All Projects",
          url: "/dashboard/projects",
        },
        {
          title: "New Project",
          url: "/dashboard/projects/new",
        },
      ],
    },
    {
      title: "Insights",
      url: "/dashboard/insights",
      icon: IconBulb,
      items: [
        {
          title: "All Insights",
          url: "/dashboard/insights",
        },
      ],
    },
    {
      title: "Tasks",
      url: "/dashboard/tasks",
      icon: IconInnerShadowTop,
    },
    {
      title: "Sources",
      url: "/dashboard/sources",
      icon: IconLink,
    },
    {
      title: "Connectors",
      url: "/dashboard/connectors",
      icon: IconPlug,
    },
    {
      title: "Skills",
      url: "/dashboard/skills",
      icon: IconBook,
    },
    {
      title: "Uploads",
      url: "/dashboard/uploads",
      icon: IconUpload,
    },
    {
      title: "Profile",
      url: "/dashboard/profile",
      icon: IconUsers,
    },
  ],
  navClouds: [
    {
      title: "Content",
      icon: IconFileDescription,
      isActive: true,
      url: "#",
      items: [
        {
          title: "Draft Articles",
          url: "/dashboard/blog?status=draft",
        },
        {
          title: "Published Articles",
          url: "/dashboard/blog?status=published",
        },
      ],
    },
    {
      title: "Media",
      icon: IconCamera,
      url: "/dashboard/uploads",
      items: [
        {
          title: "Images",
          url: "/dashboard/uploads?type=images",
        },
        {
          title: "Documents",
          url: "/dashboard/uploads?type=documents",
        },
      ],
    },
  ],
  documents: [
    {
      name: "Articles",
      url: "/dashboard/blog",
      icon: IconFileDescription,
      items: [
        {
          title: "All Articles",
          url: "/dashboard/blog",
        },
        {
          title: "Draft Articles",
          url: "/dashboard/blog?status=draft",
        },
        {
          title: "Published Articles",
          url: "/dashboard/blog?status=published",
        },
        {
          title: "New Article",
          url: "/dashboard/blog/new",
        },
      ],
    },
    {
      name: "Uploads",
      url: "/dashboard/uploads",
      icon: IconUpload,
    },
  ],
}

function isWritingPath(pathname: string) {
  return pathname === "/dashboard" || pathname === "/dashboard/"
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { state } = useSidebar()
  const location = useLocation()
  const navigate = useNavigate()
  const isWriting = isWritingPath(location.pathname)

  const startWriting = () => {
    if (isWriting) {
      window.dispatchEvent(new Event("writing-session-reset"))
      return
    }
    void navigate({ to: "/dashboard" })
  }
  
  // Fetch articles for the sidebar with infinite scrolling
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['sidebar-articles', 'all'],
    queryFn: ({ pageParam = 1 }) => getArticles(pageParam, null, 'all', 20),
    getNextPageParam: (lastPage, allPages) => {
      const currentPage = allPages.length;
      return currentPage < lastPage.total_pages ? currentPage + 1 : undefined;
    },
    initialPageParam: 1,
  });

  // Flatten all articles from all pages
  const allArticles = data?.pages.flatMap(page => page.articles) || [];

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="gap-2 px-2 pt-3 pb-1 group-data-[collapsible=icon]:px-1">
        <div className="flex h-10 items-center gap-1.5 group-data-[collapsible=icon]:justify-center">
          <SidebarTrigger className="size-8 shrink-0 text-sidebar-foreground/80 hover:bg-sidebar-accent" />
          {state === "expanded" && (
            <button
              type="button"
              onClick={startWriting}
              className="font-wordmark text-[1.35rem] leading-none tracking-tight text-sidebar-foreground"
            >
              Copilot
            </button>
          )}
        </div>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              type="button"
              tooltip="New"
              isActive={isWriting}
              onClick={startWriting}
              className="h-9 rounded-xl bg-sidebar-accent px-3 font-medium text-sidebar-foreground shadow-none hover:bg-sidebar-accent/80 data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground"
            >
              <Plus />
              <span>New</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navigationData.navMain} />
        <NavDocuments 
          articles={allArticles}
          fetchNextPage={fetchNextPage}
          hasNextPage={hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          />
      </SidebarContent>
    </Sidebar>
  )
}
