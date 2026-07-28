/**
 * Supported social media platforms with icon mappings
 */
export const SOCIAL_PLATFORMS = [
  'github',
  'linkedin',
  'twitter',
  'x',
  'discord',
  'youtube',
  'instagram',
  'facebook',
  'tiktok',
  'twitch',
  'mastodon',
  'bluesky',
  'website',
] as const;

export type SocialPlatform = (typeof SOCIAL_PLATFORMS)[number];

/**
 * A social link entry
 */
export interface SocialLink {
  platform: SocialPlatform | string; // Allow custom platforms
  url: string;
  label?: string; // Optional custom label
}

/**
 * Platform display information for the builder UI
 */
export interface PlatformInfo {
  id: SocialPlatform;
  label: string;
  placeholder: string;
}

/**
 * Platform metadata for the builder dropdown
 */
export const PLATFORM_INFO: PlatformInfo[] = [
  { id: 'github', label: 'GitHub', placeholder: 'https://github.com/username' },
  { id: 'linkedin', label: 'LinkedIn', placeholder: 'https://linkedin.com/in/username' },
  { id: 'twitter', label: 'Twitter', placeholder: 'https://twitter.com/username' },
  { id: 'x', label: 'X', placeholder: 'https://x.com/username' },
  { id: 'discord', label: 'Discord', placeholder: 'https://discord.gg/invite' },
  { id: 'youtube', label: 'YouTube', placeholder: 'https://youtube.com/@channel' },
  { id: 'instagram', label: 'Instagram', placeholder: 'https://instagram.com/username' },
  { id: 'facebook', label: 'Facebook', placeholder: 'https://facebook.com/username' },
  { id: 'tiktok', label: 'TikTok', placeholder: 'https://tiktok.com/@username' },
  { id: 'twitch', label: 'Twitch', placeholder: 'https://twitch.tv/username' },
  { id: 'mastodon', label: 'Mastodon', placeholder: 'https://mastodon.social/@username' },
  { id: 'bluesky', label: 'Bluesky', placeholder: 'https://bsky.app/profile/username' },
  { id: 'website', label: 'Website', placeholder: 'https://example.com' },
];

/**
 * Convert Record<string, string> to SocialLink[] for backwards compatibility
 */
export function parseSocialLinks(links: Record<string, string> | null | undefined): SocialLink[] {
  if (!links) return [];
  return Object.entries(links).map(([platform, url]) => ({ platform, url }));
}

/**
 * Convert SocialLink[] back to Record<string, string> for API submission
 */
export function serializeSocialLinks(links: SocialLink[]): Record<string, string> {
  return Object.fromEntries(
    links
      .filter((l) => l.url.trim() !== '')
      .map((l) => [l.platform.toLowerCase(), l.url])
  );
}
