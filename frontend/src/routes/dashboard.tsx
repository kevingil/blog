import { useState } from 'react';
import { parsePathname } from '@tanstack/react-router';
import { Users, Settings, Shield, PenLine, ImageUp } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRouter } from '@tanstack/react-router';
import { useEffect } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { Outlet } from '@tanstack/react-router';


export const Route = createFileRoute('/dashboard')({
  component: DashboardLayout,
});

function DashboardLayout() {
  const pathname = parsePathname().join('/');
  const router = useRouter();

  const navItems = [
    { href: '/dashboard', icon: Users, label: 'Profile' },
    { href: '/dashboard/blog', icon: PenLine, label: 'Articles' },
    { href: '/dashboard/uploads', icon: ImageUp, label: 'Uploads' },
    { href: '/dashboard/general', icon: Settings, label: 'General' },
    { href: '/dashboard/security', icon: Shield, label: 'Security' },
  ];

  
  // Map current route to the index of that route in our navItems
  const getCurrentIndex = () => {
    return Math.max(
      navItems.findIndex((item) => item.href === pathname),
      0
    );
  };

  const [value, setValue] = useState<number>(getCurrentIndex());

  useEffect(() => {
    setValue(getCurrentIndex());
  }, [pathname]);


  // Updated NavContent using Shadcn Tabs
  const NavContent = () => (
    <div style={{ width: '100%', overflowX: 'auto' }}>
      <div className="flex justify-between items-center">
        <div className="text-2xl font-bold ml-4 text-semibold text-sm ">Dashboard</div>
      </div>
      <Tabs defaultValue={navItems[0].href} className="w-full">
        <TabsList>
          {navItems.map((item) => (
            <TabsTrigger key={item.href} value={item.href}>
              <item.icon className="mr-1 h-4 w-4" />
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>
        {navItems.map((item) => (
          <TabsContent key={item.href} value={item.href}>
            {/* Content for {item.label} */}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );

  return (
    <div className="flex flex-col min-h-[80dvh] max-w-7xl mx-auto w-full z-[1]">

      <div className="flex flex-1 flex-col overflow-hidden h-full">
        {/* Dashboard Tabs */}
            <NavContent />

        {/* Main content */}
        <main className="flex-1 overflow-y-auto p-0">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
