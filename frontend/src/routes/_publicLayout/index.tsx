import { HeroSection } from "@/components/home/hero";
import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';
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
