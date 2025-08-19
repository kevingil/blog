"use client";
import { Terminal, Menu } from "lucide-react";
import React from "react";
import { useState, useEffect } from "react";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "./ui/sheet";
import { Separator } from "./ui/separator";
import {
  NavigationMenu,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
} from "./ui/navigation-menu";

import { signOut, useAuth } from '@/services/auth/auth';
import { Home, LogOut } from 'lucide-react';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from "@/components/ui/button";
import { Link, useNavigate } from '@tanstack/react-router';
import { ToggleTheme } from "./home/toogle-theme";
import { siteMetadata } from '@/services/constants';
import { getAboutPage } from '@/services/user';
import { useQuery } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { tokenAtom, isAuthenticatedAtom } from '@/services/auth/auth';
import { cn } from "@/lib/utils";

interface RouteProps {
  href: string;
  label: string;
}

const title: string = siteMetadata.title;

const routeList: RouteProps[] = [
  {
    href: "/blog",
    label: "Blog",
  },
  {
    href: "/projects",
    label: "Projects",
  },
  {
    href: "/about",
    label: "About",
  },
];

export const Navbar = () => {
  const [isOpen, setIsOpen] = React.useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isAnimated, setIsAnimated] = useState(false);
  const { user, token, signOut } = useAuth();
  const isAuthenticated = useAtomValue(isAuthenticatedAtom);

  useEffect(() => {
    // Trigger animation with a delay
    const timer = setTimeout(() => {
      setIsAnimated(true);
    }, 100);
    return () => clearTimeout(timer);
  }, []);

  const { data: pageData, isLoading: aboutPageLoading } = useQuery({
    queryKey: ['aboutPage'],
    queryFn: getAboutPage,
    staleTime: 5000,
    enabled: !!isAuthenticated && !!token,
  });
  console.log("navbar pageData", pageData, aboutPageLoading);


  return (
    <header className={cn(
      "transition-all duration-300 shadow-lg border border-gray-200/10 w-full sm:w-[95%] max-w-6xl top-0 sm:top-2 mx-auto sticky z-10 rounded-b-xl sm:rounded-2xl flex justify-between items-center p-4 bg-card/20 text-card-foreground mb-6",
      isAnimated ? "card-animated" : "card-hidden"
    )}>
      <Link to="/" className="flex items-center">
        <span className={'text-xl'}>{title}</span>
      </Link>
      {/* <!-- Mobile --> */}
      <div className="flex items-center lg:hidden">
        <Sheet open={isOpen} onOpenChange={setIsOpen}>
          <SheetTrigger asChild>
            <Menu
              onClick={() => setIsOpen(!isOpen)}
              className="cursor-pointer lg:hidden"
            />
          </SheetTrigger>

          <SheetContent
            side="right"
            className={cn(
              "flex flex-col justify-between rounded-tr-2xl rounded-br-2xl bg-card/20 text-card-foreground border border-gray-200/10 shadow-lg",
              "card-animated"
            )}
          >
            <div>
              <SheetHeader className="mb-4 ml-4">
                <SheetTitle className="flex items-center">
                  <Link to="/" className="flex items-center">
                    Kevin Gil
                  </Link>
                </SheetTitle>
              </SheetHeader>

              <div className="flex flex-col gap-2">
                {routeList.map(({ href, label }) => (
                  <Button
                    key={href}
                    onClick={() => setIsOpen(false)}
                    asChild
                    variant="ghost"
                    className="justify-start text-base"
                  >
                    <Link to={href}>{label}</Link>
                  </Button>
                ))}
              </div>
            </div>

            <SheetFooter className="flex-col sm:flex-col justify-start items-start">
              <Separator className="mb-2" />

              <ToggleTheme onClick={() => setIsOpen(false)}/>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>

      {/* <!-- Desktop --> */}
      <NavigationMenu className="hidden lg:block ml-auto">
        <NavigationMenuList>
            {routeList.map(({ href, label }) => (

          <NavigationMenuItem key={href}>
              <NavigationMenuLink key={href} asChild>
                <Link to={href} className="text-base px-4 font-semibold hover:text-indigo-500 dark:hover:text-indigo-400 transition">
                  {label}
                </Link>
              </NavigationMenuLink>
          </NavigationMenuItem>
            ))}
            <NavigationMenuItem>
            <ToggleTheme />
            </NavigationMenuItem>

            <NavigationMenuItem>
              <div>
                {isAuthenticated && (
            <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
              <DropdownMenuTrigger > 
                <Avatar className="cursor-pointer size-9">
                  <AvatarImage alt={user?.name || ''} />
                  <AvatarFallback>
                    {user?.email
                      .split(' ')
                      .map((n) => n[0])
                      .join('')}
                  </AvatarFallback>
                </Avatar>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="flex flex-col gap-1">
                <DropdownMenuItem className="cursor-pointer" onClick={() => setIsMenuOpen(false)}>
                  <Link to="/dashboard" className="flex w-full items-center">
                    <Home className="mr-2 h-4 w-4" />
                    <span>Dashboard</span>
                  </Link>
                </DropdownMenuItem>
                <form onSubmit={(e) => { e.preventDefault(); signOut(); }} className="w-full">
                  <button type="submit" className="flex w-full">
                    <DropdownMenuItem className="w-full flex-1 cursor-pointer"
                    onClick={() => {
                      setIsMenuOpen(false);
                      signOut();
                    }}>
                      <LogOut className="mr-2 h-4 w-4" />
                      <span>Sign out</span>
                    </DropdownMenuItem>
                  </button>
                </form>
              </DropdownMenuContent>
            </DropdownMenu>
            )}
          </div>
            </NavigationMenuItem>

        </NavigationMenuList>
      </NavigationMenu>

    </header>
  );
};
