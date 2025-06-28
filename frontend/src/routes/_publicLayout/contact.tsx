import { useEffect, useState } from 'react';
import { Card, CardContent } from '../../components/ui/card';
import { Skeleton } from '../../components/ui/skeleton';
import { getContactPage } from '../../services/user';
import { useRef } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';

export const Route = createFileRoute('/_publicLayout/contact')({
  component: ContactPage,
});

function ContactPage() {
  const { data: pageData, isLoading } = useQuery({
    queryKey: ['contactPage'],
    queryFn: getContactPage,
    staleTime: 5000,
  });

  const [socialLinks, setSocialLinks] = useState<Record<string, string>>({});

  useEffect(() => {
    if (pageData?.social_links) {
      setSocialLinks(JSON.parse(pageData.social_links));
    }
  }, [pageData]);

  // State to control the animation
  const contactPageRef = useRef<HTMLDivElement | null>(null);
  const [animate, setAnimate] = useState(false);

  // Intersection Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setAnimate(true);
          observer.unobserve(entry.target); 
        }
      },
      {
        threshold: 0.1, 
      }
    );

    if (contactPageRef.current) {
      observer.observe(contactPageRef.current);
    }

    return () => {
      if (observer && contactPageRef.current) {
        observer.unobserve(contactPageRef.current);
      }
    };
  }, []);

  return (
    <div className={`container w-full mx-auto py-8 ${animate ? 'animate' : 'hide-down'}`} ref={contactPageRef}>
      {isLoading ? (
        <div className="container mx-auto py-8">
          <Skeleton className="h-12 w-48 mb-8" />
          <Skeleton className="h-96 w-full" />
        </div>
      ) : (
        <>
          {pageData ? (
            <>
              <h1 className="text-3xl font-bold mb-8">{pageData.title}</h1>

              <div className={`grid grid-cols-1 md:grid-cols-2 gap-8 `}>
                <Card>
                  <CardContent className="p-6">
                    <div className="prose max-w-none">
                      {pageData.content.split('\n').map((paragraph: string, index: number) => (
                        <p key={index} className="mb-4">{paragraph}</p>
                      ))}
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardContent className="p-6 space-y-6">
                    {pageData.email_address && (
                    <div>
                      <h2 className="text-xl font-semibold mb-2">Email</h2>
                      <a
                        href={`mailto:${pageData.email_address}`}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        {pageData.email_address}
                      </a>
                    </div>
                    )}

                    {Object.keys(socialLinks).length > 0 && (
                      <div>
                        <h2 className="text-xl font-semibold mb-2">Social Media</h2>
                        <div className="space-y-2">
                          {Object.entries(socialLinks).map(([platform, url]) => (
                            <div key={platform}>
                              <a
                                href={url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-blue-600 hover:text-blue-800 block"
                              >
                                {platform.charAt(0).toUpperCase() + platform.slice(1)}
                              </a>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>

              <div className="text-sm text-gray-500 mt-8 hidden">
                Last updated: {new Date(pageData.last_updated).toLocaleDateString()}
              </div>
            </>
          ) : (
            <div className="container mx-auto py-8">
              <h1 className="text-3xl font-bold mb-8">Contact</h1>
              <p>Failed to load page content.</p>
            </div>
          )}
        </>
      )}
    </div>
  );
}

