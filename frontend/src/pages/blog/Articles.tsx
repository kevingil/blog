'use client'

import { Suspense } from 'react';
import ArticlesList, { ArticlesSkeleton } from '@/components/blog/ArticleList';


export default function ArticlesPage() {
  return (
    <div className="container mx-auto">
      <h1 className="text-3xl font-bold mb-8">Blog</h1>
      <Suspense fallback={<ArticlesSkeleton />}>
        <ArticlesList
        pagination={true} />
      </Suspense>
    </div>
  );
}
