'use client';

import { useState } from 'react';
import { parsePathname } from '@tanstack/react-router';
import { Users, Settings, Shield, PenLine, ImageUp } from 'lucide-react';
import { Tabs, Tab } from '@mui/material';
import { useRouter } from '@tanstack/react-router';
import { useEffect } from 'react';
import { Box } from '@mui/material';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/dashboard')({
  component: DashboardLayout,
});

function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
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

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setValue(newValue);
    // Push selected tab's route
    router.navigate({ to: navItems[newValue].href });
  };
  

  // Updated NavContent using MUI Tabs with scrollable behavior
  const NavContent = () => (
    <Box sx={{ width: '100%', overflowX: 'auto' }}>
      <div className="flex justify-between items-center">
        <div className="text-2xl font-bold ml-4 text-semibold text-sm ">Dashboard</div>
      </div>
      <Tabs
        value={value}
        onChange={handleTabChange}
        variant="scrollable"
        aria-label="dashboard navigation tabs"
        textColor="secondary"
        indicatorColor="secondary"
        TabIndicatorProps={{
          style: {
            transition: 'all 0.3s ease-out',
          },
        }}
        sx={{
          minHeight: 48, 
          '& .MuiTab-root': {
            textTransform: 'none',
            fontSize: '0.875rem', 
            padding: '6px 16px',  
            minHeight: 48, 
          },
        }}
      >
        {navItems.map((item) => (
          <Tab
            key={item.href}
            
            icon={<item.icon className="mr-1 h-4 w-4" />}
            label={item.label}
            iconPosition="start"
            disableRipple={false}
          />
        ))}
      </Tabs>
    </Box>
  );

  return (
    <div className="flex flex-col min-h-[80dvh] max-w-7xl mx-auto w-full z-[1]">

      <div className="flex flex-1 flex-col overflow-hidden h-full">
        {/* Dashboard Tabs */}
            <NavContent />

        {/* Main content */}
        <main className="flex-1 overflow-y-auto p-0">
          {children}
          </main>
      </div>
    </div>
  );
}
