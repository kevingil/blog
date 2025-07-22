import { useEffect, useState } from 'react';
import { Card, CardContent } from '../../components/ui/card';
import { Skeleton } from '../../components/ui/skeleton';
import { getAboutPage } from '../../services/user';
import { useRef } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';

export const Route = createFileRoute('/_publicLayout/about')({
  component: AboutPage,
});

function AboutPage() {
  const { data: pageData, isLoading } = useQuery({
    queryKey: ['aboutPage'],
    queryFn: getAboutPage,
    staleTime: 5000,
  });

  const aboutPageRef = useRef<HTMLDivElement | null>(null);

  return (
    <div className="container w-full mx-auto py-8" ref={aboutPageRef}>
      {isLoading ? (
        <>
          <div className="container mx-auto py-8 w-full">
            <Skeleton className="h-12 w-48 mb-8" />
            <Skeleton className="h-96 w-full" />
          </div>
        </>
      ) : (
        <>
          {pageData ? (
            <>
              <h1 className="text-3xl font-bold mb-8">{pageData.title}</h1>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                {pageData.profile_image && (
                  <div className="md:col-span-1">
                    <Card>
                      <CardContent className="p-4">
                        <img
                          src={pageData.profile_image}
                          alt="Profile"
                          className="w-full rounded-lg"
                        />
                      </CardContent>
                    </Card>
                  </div>
                )}

                <div className={`${pageData.profile_image ? 'md:col-span-2' : 'md:col-span-3'}`}>
                  <Card>
                    <CardContent className="p-6">
                      <div className="prose max-w-none">
                        {pageData.content.split('\n').map((paragraph, index) => (
                          <p key={index} className="mb-4">{paragraph}</p>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>

              <div className="text-sm text-gray-500 mt-8 hidden">
                Last updated: {new Date(pageData.last_updated).toLocaleDateString()}
              </div>
            </>
          ) : (
            <div className="container mx-auto py-8">
              <h1 className="text-3xl font-bold mb-8">About</h1>
              <p>Failed to load page content.</p>
            </div>
          )}
        </>
      )}
    </div>
  );
}
