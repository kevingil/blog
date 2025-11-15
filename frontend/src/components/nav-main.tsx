import { type Icon } from "@tabler/icons-react"

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Plus, Sparkles } from "lucide-react"
import { cn } from "@/lib/utils"

import { Link, useLocation } from "@tanstack/react-router"

export function NavMain({
  items,
}: {
  items: {
    title: string
    url: string
    icon?: Icon
  }[]
}) {
  const location = useLocation()
  
  return (
    <SidebarGroup>
      <SidebarGroupContent className="flex flex-col gap-2">
        <SidebarMenu>
          <SidebarMenuItem className="flex flex-row items-center gap-2">
            <SidebarMenuButton
              asChild
              tooltip="AI Copilot"
              isActive={location.pathname === '/dashboard' || location.pathname === '/dashboard/'}
              className="bg-gradient-to-r from-violet-600 to-indigo-600 shadow-md text-white hover:from-violet-700 hover:to-indigo-700 active:from-violet-700 active:to-indigo-700 transition-all duration-200 flex-1"
            >
              <Link to="/dashboard">
                <Sparkles className="h-4 w-4" />
                <span>Copilot</span>
              </Link>
            </SidebarMenuButton>
            <Link to="/dashboard/blog/new">
              <Button
                size="icon"
                className={cn(
                  "outline-1 outline-gray-400 shadow-md size-8 group-data-[collapsible=icon]:opacity-0",
                  location.pathname === '/dashboard/blog/new' 
                    ? 'bg-accent text-accent-foreground' 
                    : ''
                )}
                variant="outline"
              >
                <Plus className="h-4 w-4" />
                <span className="sr-only">New Article</span>
              </Button>
            </Link>
          </SidebarMenuItem>
        </SidebarMenu>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton 
                asChild
                tooltip={item.title}
                isActive={location.pathname === item.url}
              >
                <Link to={item.url}>
                  {item.icon && <item.icon />}
                  <span>{item.title}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
