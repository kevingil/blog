import { redirect, useLocation } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { Outlet } from '@tanstack/react-router';
import { AppSidebar } from "@/components/app-sidebar"
import { SiteHeader } from "@/components/site-header"
import { UserProfileBanner } from "@/components/user-profile-banner"
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { AdminDashboardProvider } from "@/services/dashboard/dashboard"
// Preload child routes so they are discovered in route tree
import './dashboard/blog/index'
import './dashboard/blog/new'
import './dashboard/blog/edit.$blogSlug'
import './dashboard/pages/index'
import './dashboard/pages/new'
import './dashboard/pages/edit.$pageId'
import './dashboard/projects/index'
import './dashboard/projects/new'
import './dashboard/projects/edit.$projectId'

export const Route = createFileRoute('/dashboard')({
  component: DashboardLayout,
  beforeLoad: ({ context, location }) => {
    if (!context.auth || !context.auth.isAuthenticated) {
      console.log("user dashboard beforeLoad", JSON.stringify(context.auth));
      throw redirect({
        to: '/login',
        search: {
          redirect: location.href,
        },
      });
    }
  },
});

function DashboardLayout() {
  return (
    <AdminDashboardProvider>
      <SidebarProvider
        className="h-screen overflow-hidden"
        style={
          {
            "--sidebar-width": "calc(var(--spacing) * 72)",
            "--header-height": "calc(var(--spacing) * 12)",
          } as React.CSSProperties
        }
      >
        <AppSidebar variant="inset" />
        <SidebarInset className="flex flex-col overflow-hidden h-full">
          <SiteHeader />
          <div className="flex flex-1 flex-col overflow-hidden min-h-0">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </AdminDashboardProvider>
  )
}
