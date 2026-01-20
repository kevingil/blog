import { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SocialLinkIcon } from '@/components/social-link-icon';
import {
  type SocialLink,
  type SocialPlatform,
  PLATFORM_INFO,
  parseSocialLinks,
  serializeSocialLinks,
} from '@/types/social-links';

interface SocialLinksBuilderProps {
  /** Current social links as Record<string, string> from the API */
  value: Record<string, string> | null | undefined;
  /** Callback when links change, returns Record<string, string> for API */
  onChange: (links: Record<string, string>) => void;
  /** Whether the builder is disabled */
  disabled?: boolean;
}

/**
 * A builder component for managing social media links.
 * Converts between the API format (Record<string, string>) and internal format (SocialLink[]).
 */
export function SocialLinksBuilder({ value, onChange, disabled }: SocialLinksBuilderProps) {
  const [links, setLinks] = useState<SocialLink[]>(() => parseSocialLinks(value));

  const handleAddLink = () => {
    // Find a platform that hasn't been used yet
    const usedPlatforms = new Set(links.map((l) => l.platform.toLowerCase()));
    const availablePlatform = PLATFORM_INFO.find(
      (p) => !usedPlatforms.has(p.id)
    );

    const newLink: SocialLink = {
      platform: availablePlatform?.id || 'website',
      url: '',
    };
    const newLinks = [...links, newLink];
    setLinks(newLinks);
    onChange(serializeSocialLinks(newLinks));
  };

  const handleRemoveLink = (index: number) => {
    const newLinks = links.filter((_, i) => i !== index);
    setLinks(newLinks);
    onChange(serializeSocialLinks(newLinks));
  };

  const handlePlatformChange = (index: number, platform: string) => {
    const newLinks = links.map((link, i) =>
      i === index ? { ...link, platform: platform as SocialPlatform } : link
    );
    setLinks(newLinks);
    onChange(serializeSocialLinks(newLinks));
  };

  const handleUrlChange = (index: number, url: string) => {
    const newLinks = links.map((link, i) =>
      i === index ? { ...link, url } : link
    );
    setLinks(newLinks);
    onChange(serializeSocialLinks(newLinks));
  };

  const getPlaceholder = (platform: string): string => {
    const info = PLATFORM_INFO.find((p) => p.id === platform.toLowerCase());
    return info?.placeholder || 'https://...';
  };

  // Get platforms that are already in use (for disabling in dropdown)
  const usedPlatforms = new Set(links.map((l) => l.platform.toLowerCase()));

  return (
    <div className="space-y-3">
      {links.map((link, index) => (
        <div key={index} className="flex items-center gap-2">
          <Select
            value={link.platform}
            onValueChange={(value) => handlePlatformChange(index, value)}
            disabled={disabled}
          >
            <SelectTrigger className="w-[140px]">
              <SelectValue>
                <div className="flex items-center gap-2">
                  <SocialLinkIcon platform={link.platform} size={16} />
                  <span className="capitalize">{link.platform}</span>
                </div>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {PLATFORM_INFO.map((platform) => (
                <SelectItem
                  key={platform.id}
                  value={platform.id}
                  disabled={
                    usedPlatforms.has(platform.id) &&
                    platform.id !== link.platform.toLowerCase()
                  }
                >
                  <div className="flex items-center gap-2">
                    <SocialLinkIcon platform={platform.id} size={16} />
                    <span>{platform.label}</span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Input
            value={link.url}
            onChange={(e) => handleUrlChange(index, e.target.value)}
            placeholder={getPlaceholder(link.platform)}
            disabled={disabled}
            className="flex-1"
          />

          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => handleRemoveLink(index)}
            disabled={disabled}
            className="shrink-0"
          >
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        </div>
      ))}

      <div className="flex justify-end">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={handleAddLink}
        disabled={disabled || links.length >= PLATFORM_INFO.length}
      >
        <Plus className="h-4 w-4 mr-2" />
        Add Social Link
      </Button>
      </div>
    </div>
  );
}
