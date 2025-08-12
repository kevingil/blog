import { HeroSection } from "@/components/home/hero";
import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';
import { useQuery } from '@tanstack/react-query';
import { listProjects } from '@/services/projects';
import { Link } from '@tanstack/react-router';

import { Card } from "@/components/ui/card";
import GithubIcon from "@/components/icons/github-icon";
import LinkedInIcon from "@/components/icons/linkedin-icon";
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_publicLayout/')({
  component: HomePage,
});

function HomePage() {
  return (
    <div className="relative">
        <div className="relative z-10">
          <HeroSection />
          {/* {token && (
            <div className="my-4 p-4 bg-gray-100 dark:bg-gray-800 rounded">
              <p>Token: {token}</p>
              <p>isAuthenticated: {isAuthenticated ? 'true' : 'false'}</p>
            </div>
          )} */}
        <Suspense fallback={<ArticlesSkeleton />}>
        <ArticlesList
          pagination={false} />
        </Suspense>
        {/* Latest Projects */}
        <section className="mt-16">
          <div className="p-8 md:p-12">
            <div className="flex items-center justify-between mb-8">
              <div className="space-y-2">
                <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-300">
                  Projects
                </h2>
                {/* <p className="text-slate-600 dark:text-slate-400">
                  Showcasing innovative solutions and creative experiments
                </p> */}
              </div>
              <Link 
                to="/projects" 
                className="group inline-flex items-center gap-2 px-6 py-3 text-white rounded-xl transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1"
              >
                <span className="font-medium">View All</span>
                <svg className="w-4 h-4 transition-transform group-hover:translate-x-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 8l4 4m0 0l-4 4m4-4H3" />
                </svg>
              </Link>
            </div>
            <LatestProjectsGrid />
          </div>
        </section>
        
        <div className="my-16 relative z-10">
            <div className="flex flex-col sm:flex-row gap-4 mx-auto">
              <Card className="border w-full rounded-lg group">
                <a href="https://github.com/kevingil" target="_blank">
                  <div className="p-4">
                  <div className="flex items-center gap-2 my-2">
                    <GithubIcon />
                    <h3 className="font-bold">Github</h3>
                  </div>
                    <p className="mb-4">Checkout my repositories and projects.</p>
                    <p  className="font-bold text-indigo-700 group-hover:text-indigo-800 group-hover:underline">See more <i
                            className="fa-solid fa-arrow-up-right-from-square"></i></p>
                  </div>
                </a>
              </Card>
              <Card className="border w-full rounded-lg group">
                <a href="https://linkedin.com/in/kevingil" target="_blank">
                <div className="p-4">
                  <div className="flex items-center gap-2 my-2">
                    <LinkedInIcon />
                    <h3 className="font-bold">LinkedIn</h3>
                  </div>
                    <p className="mb-4">Connect and network with me.</p>
                    <p className="font-bold text-indigo-700 group-hover:text-indigo-800 group-hover:underline">Connect <i
                            className="fa-solid fa-arrow-up-right-from-square"></i></p>
                  </div>
                </a>
              </Card>
            </div>
        </div>
        </div>
    </div>
  );
}

function LatestProjectsGrid() {
  const { data, isLoading } = useQuery({
    queryKey: ['latest-projects'],
    queryFn: () => listProjects(1, 6),
  });
  const projects = data?.projects ?? [];
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="relative overflow-hidden rounded-2xl bg-white dark:bg-slate-800 shadow-lg border border-slate-200 dark:border-slate-700">
            <div className="aspect-video w-full bg-gradient-to-br from-slate-200 to-slate-300 dark:from-slate-700 dark:to-slate-600 animate-pulse" />
            <div className="p-6 space-y-3">
              <div className="h-6 bg-slate-200 dark:bg-slate-600 rounded animate-pulse" />
              <div className="space-y-2">
                <div className="h-4 bg-slate-200 dark:bg-slate-600 rounded animate-pulse" />
                <div className="h-4 bg-slate-200 dark:bg-slate-600 rounded w-3/4 animate-pulse" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }
  if (projects.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gradient-to-br from-slate-100 to-slate-200 dark:from-slate-800 dark:to-slate-700 flex items-center justify-center">
          <svg className="w-8 h-8 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        </div>
        <p className="text-slate-600 dark:text-slate-400">No projects yet. Check back soon!</p>
      </div>
    );
  }
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
      {projects.map((project, index) => (
        <div
          key={project.id}
          className="group relative"
          style={{ 
            animationDelay: `${index * 150}ms`,
            animation: 'fadeInScale 0.6s ease-out forwards'
          }}
        >
          <div className="relative overflow-hidden rounded-2xl bg-white dark:bg-slate-800 shadow-lg hover:shadow-2xl transition-all duration-500 border border-slate-200 dark:border-slate-700 hover:border-indigo-200 dark:hover:border-indigo-800 transform hover:-translate-y-2">
            {/* Gradient overlay on hover */}
            <div className="absolute inset-0 bg-gradient-to-br from-indigo-500/0 via-purple-500/0 to-pink-500/0 group-hover:from-indigo-500/5 group-hover:via-purple-500/5 group-hover:to-pink-500/5 transition-all duration-500 rounded-2xl" />
            
            <a 
              href={project.url || '#'} 
              target={project.url ? '_blank' : undefined} 
              rel="noreferrer"
              className="block relative"
            >
              {/* Image */}
              <div className="relative aspect-video overflow-hidden">
                {project.image_url ? (
                  <img 
                    src={project.image_url} 
                    alt={project.title} 
                    className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" 
                  />
                ) : (
                  <div className="w-full h-full bg-gradient-to-br from-indigo-100 via-purple-50 to-pink-100 dark:from-indigo-900/30 dark:via-purple-900/30 dark:to-pink-900/30 flex items-center justify-center">
                    <svg className="w-12 h-12 text-indigo-400 dark:text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
                    </svg>
                  </div>
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-black/20 via-transparent to-transparent" />
                
                {/* External link indicator */}
                {project.url && (
                  <div className="absolute top-4 right-4 w-8 h-8 bg-white/90 dark:bg-slate-800/90 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all duration-300 transform translate-x-2 group-hover:translate-x-0">
                    <svg className="w-4 h-4 text-slate-700 dark:text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </div>
                )}
              </div>

              {/* Content */}
              <div className="p-6">
                <h3 className="text-xl font-semibold text-slate-900 dark:text-slate-100 mb-3 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                  {project.title}
                </h3>
                <p className="text-slate-600 dark:text-slate-400 leading-relaxed line-clamp-2">
                  {project.description}
                </p>
                
                {/* Hover arrow */}
                <div className="flex items-center gap-2 mt-4 text-indigo-600 dark:text-indigo-400 font-medium opacity-0 group-hover:opacity-100 transition-all duration-300 transform translate-y-2 group-hover:translate-y-0">
                  <span className="text-sm">Explore Project</span>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 8l4 4m0 0l-4 4m4-4H3" />
                  </svg>
                </div>
              </div>
            </a>
          </div>
        </div>
      ))}
      
      <style>{`
        @keyframes fadeInScale {
          from {
            opacity: 0;
            transform: translateY(20px) scale(0.95);
          }
          to {
            opacity: 1;
            transform: translateY(0) scale(1);
          }
        }
      `}</style>
    </div>
  );
}
