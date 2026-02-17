import { Link } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";

export const FooterSection = () => {
  const [isAnimated, setIsAnimated] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setIsAnimated(true), 100);
    return () => clearTimeout(timer);
  }, []);

  return (
    <footer id="footer" className="container max-w-7xl pt-20 mt-auto mx-auto z-10 px-2 sm:px-6">
      <div className={cn(
        "bg-black/30 dark:bg-black/40 backdrop-blur-md border border-white/[0.08] rounded-2xl rounded-b-none p-8 sm:p-10",
        isAnimated ? "card-animated" : "card-hidden"
      )}>
        {/* Three-column grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-10 md:gap-12">
          {/* Left - Identity */}
          <div className="space-y-3">
            <h3 className="text-lg font-semibold tracking-tight text-white">Kevin Gil</h3>
            <p className="text-sm text-white/40 leading-relaxed">
              Software Engineer<br />in San Francisco
            </p>
          </div>

          {/* Center - Navigate */}
          <div className="space-y-4">
            <h4 className="text-xs font-semibold uppercase tracking-widest text-white/30">Navigate</h4>
            <nav className="flex flex-col gap-2.5">
              <Link to="/blog" className="text-sm text-white/50 hover:text-primary transition-colors duration-200 w-fit">
                Blog
              </Link>
              <Link to="/projects" className="text-sm text-white/50 hover:text-primary transition-colors duration-200 w-fit">
                Projects
              </Link>
              <Link to="/about" className="text-sm text-white/50 hover:text-primary transition-colors duration-200 w-fit">
                About
              </Link>
            </nav>
          </div>

          {/* Right - Connect */}
          <div className="space-y-4">
            <h4 className="text-xs font-semibold uppercase tracking-widest text-white/30">Connect</h4>
            <div className="flex flex-col gap-2.5">
              <a
                href="https://github.com/kevingil"
                target="_blank"
                className="flex items-center gap-2.5 text-sm text-white/50 hover:text-primary transition-colors duration-200 w-fit"
              >
                <svg className="w-4 h-4 shrink-0" viewBox="0 0 512 512" fill="currentColor">
                  <path d="M256 32C132.3 32 32 134.9 32 261.7c0 101.5 64.2 187.5 153.2 217.9a17.56 17.56 0 003.8.4c8.3 0 11.5-6.1 11.5-11.4 0-5.5-.2-19.9-.3-39.1a102.4 102.4 0 01-22.6 2.7c-43.1 0-52.9-33.5-52.9-33.5-10.2-26.5-24.9-33.6-24.9-33.6-19.5-13.7-.1-14.1 1.4-14.1h.1c22.5 2 34.3 23.8 34.3 23.8 11.2 19.6 26.2 25.1 39.6 25.1a63 63 0 0025.6-6c2-14.8 7.8-24.9 14.2-30.7-49.7-5.8-102-25.5-102-113.5 0-25.1 8.7-45.6 23-61.6-2.3-5.8-10-29.2 2.2-60.8a18.64 18.64 0 015-.5c8.1 0 26.4 3.1 56.6 24.1a208.21 208.21 0 01112.2 0c30.2-21 48.5-24.1 56.6-24.1a18.64 18.64 0 015 .5c12.2 31.6 4.5 55 2.2 60.8 14.3 16.1 23 36.6 23 61.6 0 88.2-52.4 107.6-102.3 113.3 8 7.1 15.2 21.1 15.2 42.5 0 30.7-.3 55.5-.3 63 0 5.4 3.1 11.5 11.4 11.5a19.35 19.35 0 004-.4C415.9 449.2 480 363.1 480 261.7 480 134.9 379.7 32 256 32z" />
                </svg>
                Github
              </a>
              <a
                href="https://linkedin.com/in/kevingil"
                target="_blank"
                className="flex items-center gap-2.5 text-sm text-white/50 hover:text-primary transition-colors duration-200 w-fit"
              >
                <svg className="w-4 h-4 shrink-0" viewBox="0 0 512 512" fill="currentColor">
                  <path d="M444.17 32H70.28C49.85 32 32 46.7 32 66.89v374.72C32 461.91 49.85 480 70.28 480h373.78c20.54 0 35.94-18.21 35.94-38.39V66.89C480.12 46.7 464.6 32 444.17 32zm-273.3 373.43h-64.18V205.88h64.18zM141 175.54h-.46c-20.54 0-33.84-15.29-33.84-34.43 0-19.49 13.65-34.42 34.65-34.42s33.85 14.82 34.31 34.42c-.01 19.14-13.31 34.43-34.66 34.43zm264.43 229.89h-64.18V296.32c0-26.14-9.34-44-32.56-44-17.74 0-28.24 12-32.91 23.69-1.75 4.2-2.22 9.92-2.22 15.76v113.66h-64.18V205.88h64.18v27.77c9.34-13.3 23.93-32.44 57.88-32.44 42.13 0 74 27.77 74 87.64z" />
                </svg>
                LinkedIn
              </a>
            </div>
          </div>
        </div>

        {/* Gradient separator */}
        <div className="h-px my-8 bg-gradient-to-r from-transparent via-white/10 to-transparent" />

        {/* Bottom row */}
        <div className="flex flex-col sm:flex-row justify-between items-center gap-3">
          <a
            target="_blank"
            href="https://github.com/kevingil/blog"
            className="text-xs text-white/25 hover:text-primary transition-colors duration-200"
          >
            kevingil/blog
          </a>
          <span className="text-xs text-white/20">
            &copy; {new Date().getFullYear()} Kevin Gil
          </span>
        </div>
      </div>
    </footer>
  );
};
