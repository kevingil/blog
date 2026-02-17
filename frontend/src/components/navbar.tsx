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
import { Button } from "./ui/button";
import { Link } from '@tanstack/react-router';
import { ToggleTheme } from "./home/toogle-theme";
import { useAtomValue } from 'jotai';
import { isAuthenticatedAtom } from '@/services/auth/auth';
import { cn } from "@/lib/utils";
import { UserMenu } from './user-menu';

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

    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

    if (prefersReducedMotion) {
      element.textContent = text
      isAnimationComplete.current = true
      return
    }

    element.textContent = text
    isAnimationComplete.current = true

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
          onUpdate: function () {
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

      ;(element as any)._glitchCleanup = () => {
        element.removeEventListener('mouseenter', startGlitch)
        element.removeEventListener('mouseleave', stopGlitch)
        if (glitchTimeline) glitchTimeline.kill()
      }
    }

    addGlitchHover()

    return () => {
      if (element) {
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
  { href: "/blog", label: "Blog" },
  { href: "/projects", label: "Projects" },
  { href: "/about", label: "About" },
];

export const Navbar = () => {
  const [isOpen, setIsOpen] = React.useState(false);
  const [scrollY, setScrollY] = useState(0);
  const [mounted, setMounted] = useState(false);
  const isAuthenticated = useAtomValue(isAuthenticatedAtom);

  useEffect(() => {
    const timer = setTimeout(() => setMounted(true), 100);
    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    const onScroll = () => setScrollY(window.scrollY);
    window.addEventListener('scroll', onScroll, { passive: true });
    onScroll(); // initial
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  const start = 0;
  const end = 120;
  const progress = Math.min(1, Math.max(0, (scrollY - start) / (end - start)));
  const atTop = scrollY < 1;

  const headerStyle = cn(
    "sticky top-0 z-50 w-full",
    mounted ? "opacity-100" : "opacity-0"
  );

  return (
    <header
      className={headerStyle}
      style={{
        background: atTop ? 'transparent' : `rgba(0, 0, 0, ${0.6 * progress})`,
        backdropFilter: atTop ? 'none' : `blur(${24 * progress}px) saturate(180%)`,
        WebkitBackdropFilter: atTop ? 'none' : `blur(${24 * progress}px) saturate(180%)`,
        boxShadow: atTop ? 'none' : '0 4px 48px -12px rgba(0,0,0,0.35), 0 16px 64px -24px rgba(0,0,0,0.2)',
        transition: 'background 0.8s cubic-bezier(0.4, 0, 0.2, 1), backdrop-filter 0.8s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.8s cubic-bezier(0.4, 0, 0.2, 1)',
      }}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-3 flex items-center justify-between">
        {/* Logo */}
        <Link
          to="/"
          className="flex items-center px-2 py-1.5 -mx-2 rounded-lg hover:bg-white/5 transition-colors"
        >
          <GlitchText
            text="KG"
            className="text-2xl font-bold font-gloria text-foreground"
          />
        </Link>

        {/* Nav - desktop */}
        <nav className="hidden lg:flex items-center gap-1">
          {routeList.map(({ href, label }) => (
            <Link
              key={href}
              to={href}
              className="px-4 py-2 rounded-lg text-sm font-medium text-foreground/70 hover:text-foreground hover:bg-white/10 transition-all duration-200"
            >
              {label}
            </Link>
          ))}

          <div className="w-px h-5 bg-foreground/20 mx-2" />

          {!isAuthenticated && (
            <ToggleTheme />
          )}

          {isAuthenticated && (
            <UserMenu variant="public" />
          )}
        </nav>

        {/* Nav - mobile */}
        <div className="flex lg:hidden items-center">
          <Sheet open={isOpen} onOpenChange={setIsOpen}>
            <SheetTrigger asChild>
              <button
                onClick={() => setIsOpen(!isOpen)}
                className="p-2 rounded-lg text-foreground/60 hover:text-foreground hover:bg-white/10 transition-colors"
              >
                <Menu className="w-5 h-5" />
              </button>
            </SheetTrigger>

            <SheetContent
              side="right"
              className="flex flex-col justify-between bg-black/60 dark:bg-black/70 backdrop-blur-2xl border-l border-white/[0.08] text-white"
            >
              <div>
                <SheetHeader className="mb-6 ml-4">
                  <SheetTitle className="flex items-center">
                    <Link to="/" className="flex items-center">
                      <GlitchText
                        text="KG"
                        className="text-xl font-bold font-gloria text-white"
                      />
                    </Link>
                  </SheetTitle>
                </SheetHeader>

                <div className="flex flex-col gap-1 px-2">
                  {routeList.map(({ href, label }) => (
                    <Button
                      key={href}
                      onClick={() => setIsOpen(false)}
                      asChild
                      variant="ghost"
                      className="justify-start text-base text-white/60 hover:text-white hover:bg-white/10 rounded-xl"
                    >
                      <Link to={href}>{label}</Link>
                    </Button>
                  ))}
                </div>
              </div>

              <SheetFooter className="flex-col sm:flex-col justify-start items-start">
                <Separator className="mb-2 bg-white/10" />
                <ToggleTheme onClick={() => setIsOpen(false)} />
              </SheetFooter>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
};
