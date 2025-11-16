import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { useAuth } from '@/services/auth/auth';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Separator } from '@/components/ui/separator';
import { 
  Home, 
  LayoutDashboard, 
  Settings, 
  LogOut,
  Moon,
  Sun,
  ChevronDown 
} from 'lucide-react';
import { useTheme } from '@/components/theme-provider';
import { cn } from '@/lib/utils';

interface UserMenuProps {
  variant: 'public' | 'dashboard';
}

export function UserMenu({ variant }: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const { user, signOut } = useAuth();
  const { theme, setTheme } = useTheme();

  const buttonLabel = variant === 'public' ? 'Dashboard' : 'Home Page';
  const navLink = variant === 'public' ? '/dashboard' : '/';
  const navIcon = variant === 'public' ? LayoutDashboard : Home;
  const NavIcon = navIcon;

  const getUserInitials = () => {
    if (!user?.email) return '??';
    return user.email
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const handleSignOut = () => {
    setIsOpen(false);
    signOut();
  };

  const toggleTheme = () => {
    setTheme(theme === 'light' ? 'dark' : 'light');
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className="flex items-center gap-2 h-9 px-3"
        >
          <span className="text-sm font-medium">{buttonLabel}</span>
          <Avatar className="size-6">
            <AvatarImage alt={user?.name || ''} />
            <AvatarFallback className="text-xs">
              {getUserInitials()}
            </AvatarFallback>
          </Avatar>
          <ChevronDown className="size-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-3">
        {/* User Info Section */}
        <div className="flex items-start gap-3 mb-3">
          <Avatar className="size-10">
            <AvatarImage alt={user?.name || ''} />
            <AvatarFallback className="text-sm">
              {getUserInitials()}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">
              {user?.name || 'User'}
            </p>
            <p className="text-xs text-muted-foreground truncate">
              {user?.email || ''}
            </p>
          </div>
        </div>

        <Separator className="my-2" />

        {/* Menu Items */}
        <div className="flex flex-col gap-1">
          {/* Navigation Link */}
          <Link to={navLink} onClick={() => setIsOpen(false)}>
            <Button
              variant="ghost"
              className="w-full justify-start h-9 px-2"
            >
              <NavIcon className="mr-2 h-4 w-4" />
              <span>{variant === 'public' ? 'Dashboard' : 'Home Page'}</span>
            </Button>
          </Link>

          {/* Settings Link */}
          <Link 
            to={variant === 'dashboard' ? '/dashboard/settings' : '/dashboard/settings'} 
            onClick={() => setIsOpen(false)}
          >
            <Button
              variant="ghost"
              className="w-full justify-start h-9 px-2"
            >
              <Settings className="mr-2 h-4 w-4" />
              <span>Account Preferences</span>
            </Button>
          </Link>

          {/* Theme Toggle */}
          <Button
            variant="ghost"
            className="w-full justify-start h-9 px-2"
            onClick={toggleTheme}
          >
            {theme === 'light' ? (
              <>
                <Moon className="mr-2 h-4 w-4" />
                <span>Dark Mode</span>
              </>
            ) : (
              <>
                <Sun className="mr-2 h-4 w-4" />
                <span>Light Mode</span>
              </>
            )}
          </Button>

          <Separator className="my-1" />

          {/* Sign Out */}
          <Button
            variant="ghost"
            className="w-full justify-start h-9 px-2 text-destructive hover:text-destructive"
            onClick={handleSignOut}
          >
            <LogOut className="mr-2 h-4 w-4" />
            <span>Sign out</span>
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

