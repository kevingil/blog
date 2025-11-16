import { Menu } from "lucide-react";
import React from "react";
import { useState, useEffect, useRef } from "react";
import { gsap } from 'gsap';
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
import { Button } from "./ui/button";
import { useAuth } from '@/services/auth/auth';
import { Link } from '@tanstack/react-router';
import { ToggleTheme } from "./home/toogle-theme";
import { getAboutPage } from '@/services/user';
import { useQuery } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { isAuthenticatedAtom } from '@/services/auth/auth';
import { cn } from "@/lib/utils";
import { UserMenu } from './user-menu';

// Glitch animation component for KG title
function GlitchText({ 
  text, 
  className = ""
}: { 
  text: string
  className?: string
}) {
  const textRef = useRef<HTMLSpanElement>(null)
  const isAnimationComplete = useRef(false)

  useEffect(() => {
    const element = textRef.current
    if (!element) return

    // Check for reduced motion preference
    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    
    if (prefersReducedMotion) {
      element.textContent = text
      isAnimationComplete.current = true
      return
    }

    // Initial setup
    element.textContent = text
    isAnimationComplete.current = true

    // Add glitch hover effect
    const addGlitchHover = () => {
      const originalText = text
      let glitchTimeline: gsap.core.Timeline | null = null

      const startGlitch = () => {
        if (glitchTimeline) glitchTimeline.kill()
        
        glitchTimeline = gsap.timeline({ repeat: -1 })
        glitchTimeline.to(element, {
          duration: 0.1,
          repeat: 8,
          yoyo: true,
          ease: "power2.inOut",
          textShadow: "2px 0 #ff0000, -2px 0 #00ff00",
          filter: "blur(1px)",
          onUpdate: function() {
            if (Math.random() < 0.5) {
              const glitchChars = "!@#$%^&*(){}[]|\\:;<>?/~`"
              let glitchText = ""
              for (let i = 0; i < originalText.length; i++) {
                glitchText += Math.random() < 0.1 
                  ? glitchChars[Math.floor(Math.random() * glitchChars.length)]
                  : originalText[i]
              }
              element.textContent = glitchText
            }
          }
        })
      }
      
      const stopGlitch = () => {
        if (glitchTimeline) {
          glitchTimeline.kill()
          gsap.set(element, {
            textShadow: "none",
            filter: "blur(0px)"
          })
          element.textContent = originalText
        }
      }
      
      element.addEventListener('mouseenter', startGlitch)
      element.addEventListener('mouseleave', stopGlitch)
      
      // Store cleanup function
      ;(element as any)._glitchCleanup = () => {
        element.removeEventListener('mouseenter', startGlitch)
        element.removeEventListener('mouseleave', stopGlitch)
        if (glitchTimeline) glitchTimeline.kill()
      }
    }
    
    addGlitchHover()

    // Cleanup function
    return () => {
      if (element) {
        // Clean up glitch hover if it exists
        if ((element as any)._glitchCleanup) {
          (element as any)._glitchCleanup()
        }
        gsap.set(element, { clearProps: "all" })
      }
    }
  }, [text])

  return (
    <span 
      ref={textRef} 
      className={`${className} cursor-pointer`}
    />
  )
}

interface RouteProps {
  href: string;
  label: string;
}



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
  const [isAnimated, setIsAnimated] = useState(false);
  const { token } = useAuth();
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
        <GlitchText 
          text="KG" 
          className="text-2xl font-bold font-gloria" 
        />
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
                    <GlitchText 
                      text="KG" 
                      className="text-xl font-bold font-gloria" 
                    />
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
            {!isAuthenticated && (
              <NavigationMenuItem>
                <ToggleTheme />
              </NavigationMenuItem>
            )}

            <NavigationMenuItem>
              <div>
                {isAuthenticated && (
                  <UserMenu variant="public" />
                )}
              </div>
            </NavigationMenuItem>

        </NavigationMenuList>
      </NavigationMenu>

    </header>
  );
};
