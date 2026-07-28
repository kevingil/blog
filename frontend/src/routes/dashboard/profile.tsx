import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { SettingsSkeleton } from '@/components/settingsLoading';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
import { Pencil, X, Building2, User, Check, Plus, LogOut } from 'lucide-react';
import { SocialLinksBuilder } from '@/components/social-links-builder';
import { createFileRoute } from '@tanstack/react-router';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { 
  getMyProfile, 
  updateProfile, 
  getSiteSettings, 
  updateSiteSettings,
  UserProfile, 
  SiteSettings 
} from '@/services/profile';
import { 
  listOrganizations, 
  createOrganization, 
  joinOrganization, 
  leaveOrganization,
  Organization,
  OrganizationCreateRequest
} from '@/services/organization';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

export const Route = createFileRoute('/dashboard/profile')({
  component: ProfileSettings,
});

function ProfileSettings() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [siteSettings, setSiteSettings] = useState<SiteSettings | null>(null);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [editingProfile, setEditingProfile] = useState(false);
  const [showCreateOrg, setShowCreateOrg] = useState(false);
  const [editSocialLinks, setEditSocialLinks] = useState<Record<string, string>>({});
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Profile");
  }, [setPageTitle]);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    setIsLoading(true);
    try {
      const [profileData, settingsData, orgsData] = await Promise.all([
        getMyProfile(),
        getSiteSettings(),
        listOrganizations(),
      ]);
      setProfile(profileData);
      setSiteSettings(settingsData);
      setOrganizations(orgsData);
    } catch (error) {
      console.error('Error loading profile data:', error);
      toast({
        title: "Error",
        description: "Failed to load profile data",
        variant: "destructive",
      });
    }
    setIsLoading(false);
  }

  async function handleProfileSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.target as HTMLFormElement);
    setIsSaving(true);
    
    try {
      const data = {
        name: formData.get('name') as string,
        bio: formData.get('bio') as string,
        profile_image: formData.get('profileImage') as string,
        email_public: formData.get('emailPublic') as string,
        social_links: editSocialLinks,
        meta_description: formData.get('metaDescription') as string,
      };

      const updated = await updateProfile(data);
      if (updated) {
        setProfile(updated);
        setEditingProfile(false);
        toast({
          title: "Success",
          description: "Profile updated successfully",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to update profile",
        variant: "destructive",
      });
    }
    setIsSaving(false);
  }

  async function handleSetPublicProfile(type: 'user' | 'organization', id?: string) {
    setIsSaving(true);
    try {
      const data = {
        public_profile_type: type,
        ...(type === 'user' && profile ? { public_user_id: profile.id } : {}),
        ...(type === 'organization' && id ? { public_organization_id: id } : {}),
      };
      
      const updated = await updateSiteSettings(data);
      if (updated) {
        setSiteSettings(updated);
        toast({
          title: "Success",
          description: "Public profile updated",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to update public profile",
        variant: "destructive",
      });
    }
    setIsSaving(false);
  }

  async function handleCreateOrg(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.target as HTMLFormElement);
    setIsSaving(true);
    
    try {
      const data: OrganizationCreateRequest = {
        name: formData.get('orgName') as string,
        slug: formData.get('orgSlug') as string || undefined,
        bio: formData.get('orgBio') as string || undefined,
      };

      const newOrg = await createOrganization(data);
      setOrganizations(prev => [...prev, newOrg]);
      setShowCreateOrg(false);
      
      // Auto-join the created organization
      await joinOrganization(newOrg.id);
      setProfile(prev => prev ? { ...prev, organization_id: newOrg.id } : null);
      
      toast({
        title: "Success",
        description: "Organization created and joined",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to create organization",
        variant: "destructive",
      });
    }
    setIsSaving(false);
  }

  async function handleJoinOrg(orgId: string) {
    setIsSaving(true);
    try {
      await joinOrganization(orgId);
      setProfile(prev => prev ? { ...prev, organization_id: orgId } : null);
      toast({
        title: "Success",
        description: "Joined organization",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to join organization",
        variant: "destructive",
      });
    }
    setIsSaving(false);
  }

  async function handleLeaveOrg() {
    setIsSaving(true);
    try {
      await leaveOrganization();
      setProfile(prev => prev ? { ...prev, organization_id: undefined } : null);
      toast({
        title: "Success",
        description: "Left organization",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to leave organization",
        variant: "destructive",
      });
    }
    setIsSaving(false);
  }

  if (isLoading) {
    return <SettingsSkeleton />;
  }

  const currentOrg = organizations.find(org => org.id === profile?.organization_id);

  return (
    <section className="flex-1 p-0 md:p-4 space-y-8">
      <h1 className="text-lg lg:text-2xl font-medium mb-6">Profile</h1>
      
      {/* User Profile Card */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Your Profile</CardTitle>
            <CardDescription>Manage your personal profile information</CardDescription>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              if (!editingProfile && profile) {
                setEditSocialLinks(profile.social_links || {});
              }
              setEditingProfile(!editingProfile);
            }}
          >
            {editingProfile ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
          </Button>
        </CardHeader>
        <CardContent>
          {editingProfile ? (
            <form onSubmit={handleProfileSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Display Name</Label>
                <Input
                  id="name"
                  name="name"
                  defaultValue={profile?.name || ''}
                  required
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="bio">Bio</Label>
                <Textarea
                  id="bio"
                  name="bio"
                  rows={4}
                  defaultValue={profile?.bio || ''}
                  placeholder="Tell us about yourself..."
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="profileImage">Profile Image URL</Label>
                <Input
                  id="profileImage"
                  name="profileImage"
                  defaultValue={profile?.profile_image || ''}
                  placeholder="https://..."
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="emailPublic">Public Email</Label>
                <Input
                  id="emailPublic"
                  name="emailPublic"
                  type="email"
                  defaultValue={profile?.email_public || ''}
                  placeholder="public@example.com"
                />
              </div>
              
              <div className="space-y-2">
                <Label>Social Links</Label>
                <SocialLinksBuilder
                  value={editSocialLinks}
                  onChange={setEditSocialLinks}
                  disabled={isSaving}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="metaDescription">Meta Description</Label>
                <Textarea
                  id="metaDescription"
                  name="metaDescription"
                  rows={2}
                  defaultValue={profile?.meta_description || ''}
                  placeholder="SEO description..."
                />
              </div>
              
              <Button type="submit" disabled={isSaving}>
                {isSaving ? 'Saving...' : 'Save Profile'}
              </Button>
            </form>
          ) : (
            <div className="space-y-4">
              <div className="flex items-center gap-4">
                {profile?.profile_image && (
                  <img 
                    src={profile.profile_image} 
                    alt="Profile" 
                    className="w-16 h-16 rounded-full object-cover"
                  />
                )}
                <div>
                  <p className="font-medium text-lg">{profile?.name}</p>
                  {profile?.email_public && (
                    <p className="text-sm text-muted-foreground">{profile.email_public}</p>
                  )}
                </div>
              </div>
              
              {profile?.bio && (
                <div className="space-y-1">
                  <Label className="text-muted-foreground">Bio</Label>
                  <p className="text-sm whitespace-pre-wrap">{profile.bio}</p>
                </div>
              )}
              
              {profile?.social_links && Object.keys(profile.social_links).length > 0 && (
                <div className="space-y-1">
                  <Label className="text-muted-foreground">Social Links</Label>
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(profile.social_links).map(([platform, url]) => (
                      <a
                        key={platform}
                        href={url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-blue-600 hover:underline"
                      >
                        {platform}
                      </a>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Organization Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Building2 className="h-5 w-5" />
            Organization
          </CardTitle>
          <CardDescription>
            {currentOrg 
              ? `You are a member of ${currentOrg.name}`
              : 'Join or create an organization'
            }
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {currentOrg ? (
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div className="flex items-center gap-3">
                {currentOrg.logo_url && (
                  <img 
                    src={currentOrg.logo_url} 
                    alt={currentOrg.name} 
                    className="w-10 h-10 rounded object-cover"
                  />
                )}
                <div>
                  <p className="font-medium">{currentOrg.name}</p>
                  <p className="text-sm text-muted-foreground">@{currentOrg.slug}</p>
                </div>
              </div>
              <Button 
                variant="outline" 
                size="sm"
                onClick={handleLeaveOrg}
                disabled={isSaving}
              >
                <LogOut className="h-4 w-4 mr-2" />
                Leave
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {organizations.length > 0 && (
                <div className="space-y-2">
                  <Label>Join an existing organization</Label>
                  <div className="space-y-2">
                    {organizations.map(org => (
                      <div 
                        key={org.id}
                        className="flex items-center justify-between p-3 border rounded-lg"
                      >
                        <div>
                          <p className="font-medium">{org.name}</p>
                          <p className="text-sm text-muted-foreground">@{org.slug}</p>
                        </div>
                        <Button 
                          variant="outline" 
                          size="sm"
                          onClick={() => handleJoinOrg(org.id)}
                          disabled={isSaving}
                        >
                          Join
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              
              <Dialog open={showCreateOrg} onOpenChange={setShowCreateOrg}>
                <DialogTrigger asChild>
                  <Button variant="outline">
                    <Plus className="h-4 w-4 mr-2" />
                    Create Organization
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Create Organization</DialogTitle>
                    <DialogDescription>
                      Create a new organization to represent your team or company.
                    </DialogDescription>
                  </DialogHeader>
                  <form onSubmit={handleCreateOrg} className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="orgName">Organization Name</Label>
                      <Input
                        id="orgName"
                        name="orgName"
                        required
                        placeholder="Acme Inc."
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="orgSlug">Slug (optional)</Label>
                      <Input
                        id="orgSlug"
                        name="orgSlug"
                        placeholder="acme-inc"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="orgBio">Description (optional)</Label>
                      <Textarea
                        id="orgBio"
                        name="orgBio"
                        rows={3}
                        placeholder="Tell us about your organization..."
                      />
                    </div>
                    <DialogFooter>
                      <Button type="submit" disabled={isSaving}>
                        {isSaving ? 'Creating...' : 'Create'}
                      </Button>
                    </DialogFooter>
                  </form>
                </DialogContent>
              </Dialog>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Public Profile Settings Card */}
      <Card>
        <CardHeader>
          <CardTitle>Public Profile Settings</CardTitle>
          <CardDescription>
            Choose which profile to display on the public /about page
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Current Status */}
          {siteSettings?.public_user_id === profile?.id && siteSettings?.public_profile_type === 'user' ? (
            <div className="p-4 border rounded-lg bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800">
              <div className="flex items-center gap-2">
                <Check className="h-4 w-4 text-green-600" />
                <span className="text-sm text-green-700 dark:text-green-300">
                  Your profile is currently shown on the /about page
                </span>
              </div>
            </div>
          ) : siteSettings?.public_organization_id && siteSettings?.public_profile_type === 'organization' ? (
            <div className="p-4 border rounded-lg bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
              <div className="flex items-center gap-2">
                <Building2 className="h-4 w-4 text-blue-600" />
                <span className="text-sm text-blue-700 dark:text-blue-300">
                  Organization profile is shown on the /about page
                </span>
              </div>
            </div>
          ) : (
            <div className="p-4 border rounded-lg bg-yellow-50 dark:bg-yellow-950 border-yellow-200 dark:border-yellow-800">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4 text-yellow-600" />
                <span className="text-sm text-yellow-700 dark:text-yellow-300">
                  No public profile is set. The /about page will show a placeholder.
                </span>
              </div>
            </div>
          )}

          {/* Set User Profile as Public */}
          <div className="space-y-2">
            <Label>User Profile</Label>
            <div className="flex items-center gap-2">
              <Button
                onClick={() => handleSetPublicProfile('user')}
                disabled={isSaving || (siteSettings?.public_user_id === profile?.id && siteSettings?.public_profile_type === 'user')}
                variant={siteSettings?.public_user_id === profile?.id && siteSettings?.public_profile_type === 'user' ? 'secondary' : 'default'}
              >
                <User className="h-4 w-4 mr-2" />
                {siteSettings?.public_user_id === profile?.id && siteSettings?.public_profile_type === 'user' 
                  ? 'Currently Active' 
                  : 'Set My Profile as Public'}
              </Button>
            </div>
          </div>

          {/* Set Organization Profile as Public */}
          {organizations.length > 0 && (
            <div className="space-y-2">
              <Label>Organization Profile</Label>
              <div className="flex items-center gap-2">
                <Select
                  value={siteSettings?.public_profile_type === 'organization' ? siteSettings.public_organization_id || '' : ''}
                  onValueChange={(id) => handleSetPublicProfile('organization', id)}
                  disabled={isSaving}
                >
                  <SelectTrigger className="w-[200px]">
                    <SelectValue placeholder="Select organization" />
                  </SelectTrigger>
                  <SelectContent>
                    {organizations.map(org => (
                      <SelectItem key={org.id} value={org.id}>
                        {org.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {siteSettings?.public_profile_type === 'organization' && siteSettings?.public_organization_id && (
                  <span className="text-sm text-muted-foreground">(Currently active)</span>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  );
}
