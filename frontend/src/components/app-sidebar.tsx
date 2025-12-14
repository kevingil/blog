import * as React from "react"
import {
  IconCamera,
  IconFileDescription,
  IconFileWord,
  IconFolder,
  IconInnerShadowTop,
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
  useSidebar,
} from "@/components/ui/sidebar"
import { Link } from "@tanstack/react-router"
import { useInfiniteQuery } from "@tanstack/react-query"
import { getArticles } from "@/services/blog"


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

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { state } = useSidebar()
  
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
      <SidebarHeader>
        {state === "expanded" && (
          <div className="text-base font-semibold">Dashboard</div>
        )}
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
