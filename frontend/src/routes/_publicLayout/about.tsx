import { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card';
import { Skeleton } from '../../components/ui/skeleton';

import { Button } from '../../components/ui/button';
import { getAboutPage, getContactPage } from '../../services/user';
import { useRef } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Mail, ExternalLink } from 'lucide-react';
import GithubIcon from '../../components/icons/github-icon';
import LinkedinIcon from '../../components/icons/linkedin-icon';
import XIcon from '../../components/icons/x-icon';
import DiscordIcon from '../../components/icons/discord-icon';

export const Route = createFileRoute('/_publicLayout/about')({
  component: AboutPage,
});

function AboutPage() {
  const { data: aboutData, isLoading: aboutLoading } = useQuery({
    queryKey: ['aboutPage'],
    queryFn: getAboutPage,
    staleTime: 5000,
  });

  const { data: contactData, isLoading: contactLoading } = useQuery({
    queryKey: ['contactPage'],
    queryFn: getContactPage,
    staleTime: 5000,
  });

  const [socialLinks, setSocialLinks] = useState<Record<string, string>>({});
  const aboutPageRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (contactData?.social_links) {
      try {
        setSocialLinks(JSON.parse(contactData.social_links));
      } catch (e) {
        console.error('Error parsing social links:', e);
      }
    }
  }, [contactData]);

  const isLoading = aboutLoading || contactLoading;

  const getSocialIcon = (platform: string) => {
    const platformLower = platform.toLowerCase();
    switch (platformLower) {
      case 'github':
        return <GithubIcon className="h-5 w-5" />;
      case 'linkedin':
        return <LinkedinIcon className="h-5 w-5" />;
      case 'x':
      case 'twitter':
        return <XIcon className="h-5 w-5" />;
      case 'discord':
        return <DiscordIcon className="h-5 w-5" />;
      default:
        return <ExternalLink className="h-5 w-5" />;
    }
  };



  return (
    <div className="container w-full mx-auto py-8 space-y-8" ref={aboutPageRef}>
      {isLoading ? (
        <div className="space-y-8">
          <Skeleton className="h-12 w-64 mx-auto" />
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <Skeleton className="h-96 lg:col-span-1" />
            <Skeleton className="h-96 lg:col-span-2" />
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Skeleton className="h-64" />
            <Skeleton className="h-64" />
          </div>
        </div>
      ) : (
        <>
          {/* Main Content - Centered Layout */}
          <div className="max-w-4xl mx-auto">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {/* Profile Image */}
              {aboutData?.profile_image && (
                <div className="md:col-span-1">
                  <Card className="overflow-hidden p-0">
                    <CardContent className="p-0">
                      <div className="relative">
                        <img
                          src={aboutData.profile_image}
                          alt="Profile"
                          className="w-full aspect-square object-cover"
                        />
                        <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
                      </div>
                    </CardContent>
                  </Card>
                </div>
              )}

              {/* Bio Content */}
              <div className={`${aboutData?.profile_image ? 'md:col-span-2' : 'md:col-span-3'}`}>
                <Card>
                  <CardHeader>
                    <CardTitle>About</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="prose prose-gray max-w-none dark:prose-invert">
                      {aboutData?.content ? (
                        aboutData.content.split('\n').map((paragraph, index) => (
                          paragraph.trim() && (
                            <p key={index} className="mb-4 leading-relaxed">
                              {paragraph}
                            </p>
                          )
                        ))
                      ) : (
                        <p className="text-muted-foreground">
                          Passionate software engineer specializing in AI agents and autonomous systems.
                          I build intelligent applications that push the boundaries of what's possible.
                        </p>
                      )}
                    </div>
                  </CardContent>
                </Card>
                {/* Contact Information */}
            {contactData && (
              <Card className="mt-8">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Mail className="h-5 w-5" />
                    Get in Touch
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* Email */}
                    {contactData.email_address && (
                      <div className="space-y-2">
                        <h3 className="font-medium">Email</h3>
                        <a
                          href={`mailto:${contactData.email_address}`}
                          className="text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-2"
                        >
                          <Mail className="h-4 w-4" />
                          {contactData.email_address}
                        </a>
                      </div>
                    )}

                    {/* Social Links */}
                    {Object.keys(socialLinks).length > 0 && (
                      <div className="space-y-2">
                        <div className="flex flex-wrap gap-2">
                          {Object.entries(socialLinks).map(([platform, url]) => (
                            <Button
                              key={platform}
                              variant="outline"
                              size="sm"
                              asChild
                              className="flex items-center gap-2"
                            >
                              <a
                                href={url}
                                target="_blank"
                                rel="noopener noreferrer"
                              >
                                {getSocialIcon(platform)}
                                {platform.charAt(0).toUpperCase() + platform.slice(1)}
                              </a>
                            </Button>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            )}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
