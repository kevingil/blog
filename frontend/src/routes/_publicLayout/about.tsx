import { useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card';
import { Skeleton } from '../../components/ui/skeleton';
import { Button } from '../../components/ui/button';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Mail, Globe } from 'lucide-react';
import { SocialLinkIcon } from '../../components/social-link-icon';
import { getPublicProfile } from '../../services/profile';

export const Route = createFileRoute('/_publicLayout/about')({
  component: AboutPage,
});

function AboutPage() {
  const { data: profile, isLoading } = useQuery({
    queryKey: ['publicProfile'],
    queryFn: getPublicProfile,
    staleTime: 5000,
  });

  const aboutPageRef = useRef<HTMLDivElement | null>(null);

  const socialLinks = profile?.social_links || {};
  const hasSocialLinks = Object.keys(socialLinks).length > 0;

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
      ) : !profile ? (
        <div className="max-w-4xl mx-auto">
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-muted-foreground">
                No public profile has been configured yet.
              </p>
            </CardContent>
          </Card>
        </div>
      ) : (
        <>
          {/* Main Content - Centered Layout */}
          <div className="max-w-4xl mx-auto">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {/* Profile Image / Logo */}
              {profile.image_url && (
                <div className="md:col-span-1">
                  <Card className="overflow-hidden p-0">
                    <CardContent className="p-0">
                      <div className="relative">
                        <img
                          src={profile.image_url}
                          alt={profile.type === 'organization' ? 'Logo' : 'Profile'}
                          className="w-full aspect-square object-cover"
                        />
                        <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent" />
                      </div>
                    </CardContent>
                  </Card>
                </div>
              )}

              {/* Bio Content */}
              <div className={`${profile.image_url ? 'md:col-span-2' : 'md:col-span-3'}`}>
                <Card>
                  <CardHeader>
                    <CardTitle>
                      {profile.type === 'organization' ? profile.name : 'About'}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="prose prose-gray max-w-none dark:prose-invert">
                      {profile.bio ? (
                        profile.bio.split('\n').map((paragraph, index) => (
                          paragraph.trim() && (
                            <p key={index} className="mb-4 leading-relaxed">
                              {paragraph}
                            </p>
                          )
                        ))
                      ) : (
                        <p className="text-muted-foreground">
                          {profile.type === 'organization' 
                            ? 'Learn more about our organization.'
                            : 'Passionate software engineer specializing in AI agents and autonomous systems. I build intelligent applications that push the boundaries of what\'s possible.'
                          }
                        </p>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {/* Contact Information */}
                {(profile.email_public || hasSocialLinks || profile.website_url) && (
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
                        {profile.email_public && (
                          <div className="space-y-2">
                            <h3 className="font-medium">Email</h3>
                            <a
                              href={`mailto:${profile.email_public}`}
                              className="text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-2"
                            >
                              <Mail className="h-4 w-4" />
                              {profile.email_public}
                            </a>
                          </div>
                        )}

                        {/* Website (org only) */}
                        {profile.website_url && (
                          <div className="space-y-2">
                            <h3 className="font-medium">Website</h3>
                            <a
                              href={profile.website_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-2"
                            >
                              <Globe className="h-4 w-4" />
                              {profile.website_url}
                            </a>
                          </div>
                        )}

                        {/* Social Links */}
                        {hasSocialLinks && (
                          <div className="space-y-2 md:col-span-2">
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
                                    <SocialLinkIcon platform={platform} size={20} />
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
