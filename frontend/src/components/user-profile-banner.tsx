import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/services/auth/auth';
import { getAboutPage, getContactPage, AboutPageData, ContactPageData } from '@/services/user';
import GithubIcon from '@/components/icons/github-icon';
import LinkedinIcon from '@/components/icons/linkedin-icon';
import XIcon from '@/components/icons/x-icon';
import DiscordIcon from '@/components/icons/discord-icon';
import { Mail, ExternalLink, User } from 'lucide-react';

interface SocialLink {
  platform: string;
  url: string;
}

export function UserProfileBanner() {
  const { user } = useAuth();
  
  const { data: aboutData } = useQuery<AboutPageData | null>({
    queryKey: ['aboutPage'],
    queryFn: getAboutPage,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  const { data: contactData } = useQuery<ContactPageData | null>({
    queryKey: ['contactPage'],
    queryFn: getContactPage,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  const socialLinks = useMemo(() => {
    if (!contactData?.social_links) return [];
    
    try {
      const parsed = JSON.parse(contactData.social_links);
      
      // Handle object format: {"github": "url", "linkedin": "url"}
      if (typeof parsed === 'object' && !Array.isArray(parsed)) {
        return Object.entries(parsed).map(([platform, url]) => ({
          platform,
          url: url as string
        }));
      }
      
      // Handle array format: [{"platform": "github", "url": "url"}]
      return Array.isArray(parsed) ? parsed : [];
    } catch (error) {
      console.error('Error parsing social links:', error);
      return [];
    }
  }, [contactData?.social_links]);

  const getSocialIcon = (platform: string) => {
    const platformLower = platform.toLowerCase();
    switch (platformLower) {
      case 'github':
        return <GithubIcon className="h-4 w-4" />;
      case 'linkedin':
        return <LinkedinIcon className="h-4 w-4" />;
      case 'x':
      case 'twitter':
        return <XIcon className="h-4 w-4" />;
      case 'discord':
        return <DiscordIcon className="h-4 w-4" />;
      default:
        return <ExternalLink className="h-4 w-4" />;
    }
  };

  const getUserInitials = (name: string) => {
    return name
      .split(' ')
      .map(word => word[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  return (
    <Card className="w-full">
      <CardContent className="p-6">
        <div className="flex flex-col md:flex-row gap-6 items-start">
          {/* Profile Picture */}
          <div className="flex-shrink-0">
            <Avatar className="h-24 w-24 md:h-32 md:w-32">
              <AvatarImage 
                src={aboutData?.profile_image || undefined} 
                alt={user?.name || 'Profile'} 
              />
              <AvatarFallback className="text-xl md:text-2xl">
                {user?.name ? getUserInitials(user.name) : <User className="h-8 w-8" />}
              </AvatarFallback>
            </Avatar>
          </div>

          {/* Profile Info */}
          <div className="flex-1 space-y-4">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl md:text-3xl font-bold">
                  {aboutData?.title || user?.name || 'Welcome'}
                </h1>
                <Badge variant="secondary" className="text-xs">
                  {user?.role || 'User'}
                </Badge>
              </div>
              {contactData?.email_address && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground mb-3">
                  <Mail className="h-4 w-4" />
                  {contactData.email_address}
                </div>
              )}
            </div>

            {/* About Text */}
            {aboutData?.content && (
              <div className="prose prose-sm max-w-none">
                <p className="text-muted-foreground leading-relaxed">
                  {aboutData.content}
                </p>
              </div>
            )}

            {/* Social Links */}
            {socialLinks.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2">
                {socialLinks.map((link: SocialLink, index: number) => (
                  <Button
                    key={index}
                    variant="outline"
                    size="sm"
                    asChild
                    className="h-8 px-3"
                  >
                    <a
                      href={link.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2"
                    >
                      {getSocialIcon(link.platform)}
                      <span className="capitalize">{link.platform}</span>
                    </a>
                  </Button>
                ))}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
} 
