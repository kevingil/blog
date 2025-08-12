import { HeroSection } from "@/components/home/hero";
import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';
import { useQuery } from '@tanstack/react-query';
import { listProjects } from '@/services/projects';
import { Link } from '@tanstack/react-router';
import { Card as UICard, CardContent as UICardContent } from '@/components/ui/card';
import { Card } from "@/components/ui/card";
import GithubIcon from "@/components/icons/github-icon";
import LinkedInIcon from "@/components/icons/linkedin-icon";
import { createFileRoute } from '@tanstack/react-router';
import { useAtomValue } from 'jotai';
import { tokenAtom, isAuthenticatedAtom } from '@/services/auth/auth';
import { SpiralGalaxyAnimation } from "@/components/home/galaxy";

export const Route = createFileRoute('/_publicLayout/')({
  component: HomePage,
});

function HomePage() {
  const token = useAtomValue(tokenAtom);
  const isAuthenticated = useAtomValue(isAuthenticatedAtom);
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
        <section className="mt-10">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold">Latest Projects</h2>
            <Link to="/projects" className="text-sm text-primary hover:underline">See all</Link>
          </div>
          <LatestProjectsGrid />
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
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <UICard key={i} className="h-48 animate-pulse bg-muted" />
        ))}
      </div>
    );
  }
  if (projects.length === 0) {
    return <div className="text-muted-foreground">No projects yet.</div>;
  }
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {projects.map((p) => (
        <UICard key={p.id} className="group overflow-hidden border">
          <a href={p.url || '#'} target={p.url ? '_blank' : undefined} rel="noreferrer">
            {p.image_url ? (
              <img src={p.image_url} alt={p.title} className="aspect-video w-full object-cover transition-transform duration-300 group-hover:scale-105" />
            ) : (
              <div className="aspect-video w-full bg-gray-200 dark:bg-gray-800" />
            )}
            <UICardContent className="p-4">
              <h3 className="font-semibold mb-1">{p.title}</h3>
              <p className="text-sm text-muted-foreground line-clamp-2">{p.description}</p>
            </UICardContent>
          </a>
        </UICard>
      ))}
    </div>
  );
}
