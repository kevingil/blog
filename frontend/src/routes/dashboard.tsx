import { useState, useEffect } from 'react';
import { redirect, useLocation } from '@tanstack/react-router';
import { Users, Settings, Shield, PenLine, ImageUp } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { createFileRoute, useRouter } from '@tanstack/react-router';
import { Outlet, Link } from '@tanstack/react-router';

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
      })
    }
  },
});

function DashboardContent() {

  const pathname = useLocation().pathname;

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
  console.log("dashboard value", value);

  useEffect(() => {
    setValue(getCurrentIndex());
  }, [pathname]);

  console.log("dashboard pathname", pathname);

  const NavContent = () => (
    <div style={{ width: '100%', overflowX: 'auto' }}>
      <div className="flex justify-between items-center">
        <div className="text-2xl font-medium ml-2 text-xl my-2">Dashboard</div>
      </div>
      <Tabs defaultValue={navItems.find(item => pathname === item.href)?.href || navItems[0].href} className="w-full">
        <TabsList>
          {navItems.map((item) => (
            <TabsTrigger asChild key={item.href} value={item.href}>
              <Link
                to={item.href}
                className="flex items-center"
              >
                <item.icon className="mr-1 h-4 w-4" />
                {item.label}
              </Link>
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

function DashboardLayout() {
  return <DashboardContent />;
}
