
import { HeroSection } from "@/components/home/hero";
import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';

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
    <div>
      <HeroSection />
      <Suspense fallback={<ArticlesSkeleton />}>
      <ArticlesList
        pagination={false} />
      </Suspense>
    </div>
  );
}
