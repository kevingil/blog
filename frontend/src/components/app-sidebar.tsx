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
  IconShield,
  IconPencil,
  IconUpload,
  IconHome,
} from "@tabler/icons-react"

import { NavDocuments } from "@/components/nav-documents"
import { NavMain } from "@/components/nav-main"
import { NavSecondary } from "@/components/nav-secondary"
import { NavUser } from "@/components/nav-user"
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
import { ArticleListItem } from "@/services/types"

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
  navSecondary: [
    {
      title: "Settings",
      url: "/dashboard/settings",
      icon: IconSettings,
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
        <NavSecondary items={navigationData.navSecondary} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
    </Sidebar>
  )
}
