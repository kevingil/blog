
import { useLocation } from "@tanstack/react-router"
import { PenLine } from "lucide-react"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"

import { useAdminDashboard } from "@/services/dashboard/dashboard";
import { UserMenu } from './user-menu';


export function SiteHeader() {
  const { pageTitle, blankEditor, setBlankEditor } = useAdminDashboard();
  const { pathname } = useLocation();
  const isWriting = pathname === "/dashboard" || pathname === "/dashboard/";

  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b border-border bg-background/95 text-foreground backdrop-blur supports-[backdrop-filter]:bg-background/80 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1 md:hidden" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4 md:hidden"
        />
        <h1 className="dashboard-header-title">{pageTitle}</h1>
        <div className="ml-auto flex items-center gap-2">
          {isWriting && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-9 shrink-0"
              onClick={() => setBlankEditor(!blankEditor)}
            >
              <PenLine className="h-4 w-4" />
              <span className="hidden sm:inline">
                {blankEditor ? "Switch to chat" : "Switch to editor"}
              </span>
              <span className="sm:hidden">{blankEditor ? "Chat" : "Editor"}</span>
            </Button>
          )}
          <UserMenu variant="dashboard" />
        </div>
      </div>
    </header>
  )
}
