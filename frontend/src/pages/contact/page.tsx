'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { getContactPage } from '@/db/queries';
import { useRef } from 'react';

interface ContactPage {
  id: number;
  title: string;
  content: string;
  emailAddress: string;
  socialLinks?: string;
  metaDescription?: string;
  lastUpdated: string;
}


export default function ContactPage() {
  const [pageData, setPageData] = useState<ContactPage | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [socialLinks, setSocialLinks] = useState<Record<string, string>>({});

  useEffect(() => {
    const loadData = async () => {
      try {
        const data = await getContactPage();
        if (!data) {
          return;
        }
        setPageData(data as ContactPage);
        if (data?.socialLinks) {
          setSocialLinks(JSON.parse(data.socialLinks));
        }
      } catch (error) {
        console.error('Failed to load contact page:', error);
      } finally {
        setIsLoading(false);
      }
    };
    loadData();
  }, []);

  
  // State to control the animation
  const contactPageRef = useRef<HTMLDivElement | null>(null);
  const [animate, setAnimate] = useState(false);


  // Intersection Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        console.log("entry.isIntersecting", entry.isIntersecting);
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
      console.log("contactPageRef.current", contactPageRef.current);
      observer.observe(contactPageRef.current);
    }

    return () => {
      // Clean up on unmount
      if (observer && contactPageRef.current) {
        console.log("observer.unobserve(contactPageRef.current)");
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
                      {pageData.content.split('\n').map((paragraph, index) => (
                        <p key={index} className="mb-4">{paragraph}</p>
                      ))}
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardContent className="p-6 space-y-6">
                    {pageData.emailAddress && (
                    <div>
                      <h2 className="text-xl font-semibold mb-2">Email</h2>
                      <a
                        href={`mailto:${pageData.emailAddress}`}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        {pageData.emailAddress}
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
                Last updated: {new Date(pageData.lastUpdated).toLocaleDateString()}
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

