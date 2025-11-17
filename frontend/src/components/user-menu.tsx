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
  Sun
} from 'lucide-react';
import { useTheme } from '@/components/theme-provider';

interface UserMenuProps {
  variant: 'public' | 'dashboard';
}

export function UserMenu({ variant }: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const { user, signOut } = useAuth();
  const { theme, setTheme } = useTheme();

  const buttonLabel = variant === 'public' ? 'Dashboard' : 'Home';
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
    <div className="flex items-center gap-1">
      {/* Navigation Button */}
      <Link to={navLink}>
        <Button
          variant="default"
          className="h-7 mx-2 px-2 rounded-md border-2 border-primary"
        >
          <span className="text-xs font-light">{buttonLabel}</span>
        </Button>
      </Link>

      {/* Avatar Popover */}
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="ghost"
            className="relative h-9 w-9 rounded-full p-0"
          >
            <Avatar className="size-9">
              <AvatarImage alt={user?.name || ''} />
              <AvatarFallback className="text-sm">
                {getUserInitials()}
              </AvatarFallback>
            </Avatar>
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
    </div>
  );
}

