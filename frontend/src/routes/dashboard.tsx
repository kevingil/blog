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
// Preload child routes so they are discovered in route tree
import './dashboard/blog/index'
import './dashboard/blog/new'
import './dashboard/blog/edit.$blogSlug'
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
  const location = useLocation();
  const isRootDashboard = location.pathname === '/dashboard';

  return (
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
          {isRootDashboard ? (
            <div className="@container/main flex flex-1 flex-col gap-2">
              <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
                <div className="px-4 lg:px-6">
                  <UserProfileBanner />
                </div>
              </div>
            </div>
          ) : (
            <main className="flex flex-1 flex-col overflow-hidden p-0 min-h-0">
              <Outlet />
            </main>
          )}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
