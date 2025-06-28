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

const data = {
  user: {
    name: "Blog User",
    email: "user@example.com",
    avatar: "/avatars/user.jpg",
  },
  navMain: [
    {
      title: "Profile",
      url: "/dashboard",
      icon: IconUsers,
    },
    {
      title: "Articles",
      url: "/dashboard/blog",
      icon: IconPencil,
      items: [
        {
          title: "All Articles",
          url: "/dashboard/blog",
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
      title: "General Settings",
      url: "/dashboard/general",
      icon: IconSettings,
    },
    {
      title: "Security",
      url: "/dashboard/security",
      icon: IconShield,
    },
    {
      title: "Get Help",
      url: "#",
      icon: IconHelp,
    },
  ],
  documents: [
    {
      name: "Blog Analytics",
      url: "/dashboard/analytics",
      icon: IconChartBar,
    },
    {
      name: "Content Library",
      url: "/dashboard/library",
      icon: IconDatabase,
    },
    {
      name: "Writing Assistant",
      url: "/dashboard/assistant",
      icon: IconFileAi,
    },
  ],
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5"
            >
              <a href="/">
                <IconInnerShadowTop className="!size-5" />
                <span className="text-base font-semibold">Blog Dashboard</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavDocuments items={data.documents} />
        <NavSecondary items={data.navSecondary} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={data.user} />
      </SidebarFooter>
    </Sidebar>
  )
}
