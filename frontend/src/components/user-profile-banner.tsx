import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/services/auth/auth';
import { getMyProfile, UserProfile } from '@/services/profile';
import { SocialLinkIcon } from '@/components/social-link-icon';
import { Mail, User } from 'lucide-react';

interface SocialLink {
  platform: string;
  url: string;
}

export function UserProfileBanner() {
  const { user } = useAuth();
  
  const { data: profile } = useQuery<UserProfile | null>({
    queryKey: ['myProfile'],
    queryFn: getMyProfile,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  const socialLinks = useMemo(() => {
    if (!profile?.social_links) return [];
    
    try {
      // social_links is already an object from the API
      const links = profile.social_links;
      
      // Handle object format: {"github": "url", "linkedin": "url"}
      if (typeof links === 'object' && !Array.isArray(links)) {
        return Object.entries(links).map(([platform, url]) => ({
          platform,
          url: url as string
        }));
      }
      
      return [];
    } catch (error) {
      console.error('Error parsing social links:', error);
      return [];
    }
  }, [profile?.social_links]);

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
                src={profile?.profile_image || undefined} 
                alt={profile?.name || user?.name || 'Profile'} 
              />
              <AvatarFallback className="text-xl md:text-2xl">
                {(profile?.name || user?.name) ? getUserInitials(profile?.name || user?.name || '') : <User className="h-8 w-8" />}
              </AvatarFallback>
            </Avatar>
          </div>

          {/* Profile Info */}
          <div className="flex-1 space-y-4">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl md:text-3xl font-bold">
                  {profile?.name || user?.name || 'Welcome'}
                </h1>
                <Badge variant="secondary" className="text-xs">
                  {user?.role || 'User'}
                </Badge>
              </div>
              {profile?.email_public && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground mb-3">
                  <Mail className="h-4 w-4" />
                  {profile.email_public}
                </div>
              )}
            </div>

            {/* About Text */}
            {profile?.bio && (
              <div className="prose prose-sm max-w-none">
                <p className="text-muted-foreground leading-relaxed">
                  {profile.bio}
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
                      <SocialLinkIcon platform={link.platform} size={16} />
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
