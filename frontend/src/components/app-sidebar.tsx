import * as React from "react"
import {
  IconCamera,
  IconChartBar,
  IconDashboard,
  IconDatabase,
  IconFileAi,
  IconFileDescription,
  IconFileWord,
  IconFolder,
  IconHelp,
  IconInnerShadowTop,
  IconListDetails,
  IconReport,
  IconSearch,
  IconSettings,
  IconUsers,
  IconUpload,
  IconHome,
} from "@tabler/icons-react"

import { NavDocuments } from "@/components/nav-documents"
import { NavMain } from "@/components/nav-main"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { useAuth } from "@/services/auth/auth"
import { useEffect, useState } from "react"
import { Link } from "@tanstack/react-router"
import { useInfiniteQuery } from "@tanstack/react-query"
import { getArticles } from "@/services/blog"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar"


const navigationData = {
  navMain: [
    {
      title: "Home",
      url: "/dashboard",
      icon: IconHome,
    },
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
  const { user } = useAuth();

  const [userData, setUserData] = useState<{ 
    name: string, 
    email: string,
    avatar: string,
  }>({ 
    name: "", 
    email: "", 
    avatar: "" 
  });

  // Fetch articles for the sidebar with infinite scrolling
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    error
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

  useEffect(() => {
    if (user) {
      setUserData({
        name: user.name,
        email: user.email,
        avatar: "",
      });
    }
  }, [user]);

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5"
            >
              <Link to="/">
                <IconInnerShadowTop className="!size-5" />
                <span className="text-base font-semibold">Blog Dashboard</span>
              </Link>
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
      <SidebarFooter>
        <Link to="/dashboard/settings">
        <Button
            className="bg-slate-100 text-slate-800 hover:bg-slate-200 hover:text-slate-900 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600 dark:hover:text-slate-100 w-full h-14"
            >
              <Avatar className="h-8 w-8 rounded-lg grayscale">
                <AvatarImage src={userData.avatar} alt={userData.name} />
                <AvatarFallback className="rounded-lg">CN</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{userData.name}</span>
                <span className="text-muted-foreground truncate text-xs">
                  {userData.email}
                </span>
              </div>
              <IconSettings className="h-5 w-5 text-muted-foreground" />
            </Button>
        </Link>
      </SidebarFooter>
    </Sidebar>
  )
}
