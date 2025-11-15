import { IconCirclePlusFilled, IconMail, type Icon } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { GenerateArticleDrawer } from "@/components/blog/GenerateArticleDrawer"
import { Plus, Sparkles } from "lucide-react"

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
          <SidebarMenuItem className="flex items-center gap-2">
            <GenerateArticleDrawer>
              <SidebarMenuButton
                tooltip="Quick Generate"
                className="bg-primary shadow-md text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground active:bg-primary/90 active:text-primary-foreground min-w-8 duration-200 ease-linear"
              >
                <Sparkles className="h-4 w-4" />
                <span>Quick Generate</span>
              </SidebarMenuButton>
            </GenerateArticleDrawer>
            <Link to="/dashboard/blog/new">
            <Button
              size="icon"
                className={`outline-1 outline-gray-400 shadow-md size-8 group-data-[collapsible=icon]:opacity-0 ${
                  location.pathname === '/dashboard/blog/new' 
                    ? 'bg-accent text-accent-foreground' 
                    : ''
                }`}
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
