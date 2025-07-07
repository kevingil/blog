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
import { useQuery } from "@tanstack/react-query"
import { getArticles } from "@/services/blog"
import { ArticleListItem } from "@/services/types"

const data = {
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

  // Fetch articles for the sidebar
  const { data: articlesPayload, isLoading, error } = useQuery<{ articles: ArticleListItem[], total_pages: number }>({
    queryKey: ['articles', 0, 20],
    queryFn: () => getArticles(0, null, true, 20) as Promise<{ articles: ArticleListItem[], total_pages: number }>
  });

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
        <NavMain items={data.navMain} />
        <NavDocuments articles={articlesPayload?.articles || []} />
        <NavSecondary items={data.navSecondary} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
    </Sidebar>
  )
}
