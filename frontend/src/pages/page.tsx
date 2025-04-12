
import { HeroSection } from "@/components/home/hero";
import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';
import { Card, CardContent } from "@/components/ui/card";
import GithubIcon from "@/components/icons/github-icon";
import LinkedInIcon from "@/components/icons/linkedin-icon";


export const metadata = {
  title: "Kevin Gil",
  description: "Software Engineer in San Francisco.",
  openGraph: {
    type: "website",
    url: "https://kevingil.com",
    title: "Kevin Gil",
    description: "Software Engineer in San Francisco.",
    images: [
      {
        url: "",
        width: 1200,
        height: 630,
        alt: "Kevin Gil",
      },
    ],
  },
};


export default function HomePage() {
  return (
    <div className="">
      <HeroSection />
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
  );
}
