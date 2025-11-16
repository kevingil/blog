
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"

import { isAuthenticatedAtom, useAuth } from '@/services/auth/auth';
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { useAtomValue } from "jotai";
import { getAboutPage } from "@/services/user";
import { Button } from "./ui/button";
import { useAdminDashboard } from "@/services/dashboard/dashboard";


export function SiteHeader() {
  const { token } = useAuth();
  const isAuthenticated = useAtomValue(isAuthenticatedAtom);
  const { pageTitle } = useAdminDashboard();

  const { data: pageData, isLoading: aboutPageLoading } = useQuery({
    queryKey: ['aboutPage'],
    queryFn: getAboutPage,
    staleTime: 5000,
    enabled: !!isAuthenticated && !!token,
  });
  console.log("navbar pageData", pageData, aboutPageLoading);
  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <h1 className="text-base font-medium">{pageTitle}</h1>
        <div className="ml-auto flex items-center gap-2">
        <Link to="/">
          <Button variant="outline" size="sm">
            Home
          </Button>
        </Link>
        </div>
      </div>
    </header>
  )
}
