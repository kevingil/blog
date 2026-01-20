import {
  IconBrandGithub,
  IconBrandLinkedin,
  IconBrandX,
  IconBrandDiscord,
  IconBrandYoutube,
  IconBrandInstagram,
  IconBrandFacebook,
  IconBrandTiktok,
  IconBrandTwitch,
  IconBrandMastodon,
  IconBrandBluesky,
  IconWorld,
  IconLink,
} from '@tabler/icons-react';
import type { Icon } from '@tabler/icons-react';

/**
 * Map of platform names to their corresponding Tabler icons
 */
const ICON_MAP: Record<string, Icon> = {
  github: IconBrandGithub,
  linkedin: IconBrandLinkedin,
  twitter: IconBrandX,
  x: IconBrandX,
  discord: IconBrandDiscord,
  youtube: IconBrandYoutube,
  instagram: IconBrandInstagram,
  facebook: IconBrandFacebook,
  tiktok: IconBrandTiktok,
  twitch: IconBrandTwitch,
  mastodon: IconBrandMastodon,
  bluesky: IconBrandBluesky,
  website: IconWorld,
};

interface SocialLinkIconProps {
  platform: string;
  className?: string;
  size?: number;
}

/**
 * Renders the appropriate Tabler icon for a given social platform.
 * Falls back to a generic link icon for unknown platforms.
 */
export function SocialLinkIcon({ platform, className, size = 20 }: SocialLinkIconProps) {
  const IconComponent = ICON_MAP[platform.toLowerCase()] || IconLink;
  return <IconComponent className={className} size={size} />;
}
