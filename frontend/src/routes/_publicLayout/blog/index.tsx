import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_publicLayout/blog/')({
  component: ArticlesPage,
});

export default function ArticlesPage() {
  return (
    <div className="container mx-auto">
      <h1 className="text-3xl font-bold mb-8 mt-8">Blog</h1>
      <Suspense fallback={<ArticlesSkeleton />}>
        <ArticlesList
        pagination={true} />
      </Suspense>
    </div>
  );
}
