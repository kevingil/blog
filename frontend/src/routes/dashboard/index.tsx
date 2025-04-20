import { useState, useEffect } from 'react';
import { AboutPageData, ContactPageData, getAboutPage, getContactPage, updateAboutPage, updateContactPage } from '@/services/user';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { SettingsSkeleton } from '@/components/settingsLoading';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
import { Pencil, X } from 'lucide-react';
import { createFileRoute } from '@tanstack/react-router';


export const Route = createFileRoute('/dashboard/')({
  component: Settings,
});

function Settings() {
  const [aboutData, setAboutData] = useState<AboutPageData | null>(null);
  const [contactData, setContactData] = useState<ContactPageData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [editingAbout, setEditingAbout] = useState(false);
  const [editingContact, setEditingContact] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    const loadData = async () => {
      const [about, contact] = await Promise.all([
        getAboutPage(),
        getContactPage(),
      ]);
      if (about) {
        setAboutData(about);
      }
      if (contact) {
        setContactData(contact);
      }
    };
    loadData();
  }, []);

  async function handleAboutSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.target as HTMLFormElement);
    setIsLoading(true);
    try {
      if (aboutData) {
        const data = {
          id: aboutData.id,
          title: formData.get('title') as string,
          content: formData.get('content') as string,
          profile_image: formData.get('profileImage') as string,
          meta_description: formData.get('metaDescription') as string,
          last_updated: new Date().toISOString(),
        };

      const success = await updateAboutPage(data);
      
      if (!success) throw new Error('Failed to update');
      
      setAboutData(prev => ({ ...prev!, ...data }));
      setEditingAbout(false);
      
      toast({
        title: "Success",
        description: "About page updated successfully",
      });
    } else {
      throw new Error('About data not found');
    }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to update about page",
        variant: "destructive",
      });
    }
    setIsLoading(false);
  }

  async function handleContactSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.target as HTMLFormElement);
    setIsLoading(true);
    try {
      if (contactData) {
        const data = {
          id: contactData.id,
          title: formData.get('title') as string,
          content: formData.get('content') as string,
          email_address: formData.get('emailAddress') as string,
          social_links: formData.get('socialLinks') as string,
          meta_description: formData.get('metaDescription') as string,
          last_updated: new Date().toISOString(),
        };

      const success = await updateContactPage(data);
      
      if (!success) throw new Error('Failed to update');
      
      setContactData(prev => ({ ...prev!, ...data }));
      setEditingContact(false);
      
      toast({
        title: "Success",
        description: "Contact page updated successfully",
      });
    } else {
      throw new Error('Contact data not found');
    }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to update contact page",
        variant: "destructive",
      });
    }
    setIsLoading(false);
  }

  if (!aboutData || !contactData) {
    return <SettingsSkeleton />;
  }

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium mb-6">Profile</h1>
      
      <Card className="mb-8">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>About Page</CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setEditingAbout(!editingAbout)}
          >
            {editingAbout ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
          </Button>
        </CardHeader>
        <CardContent>
          {editingAbout ? (
            <form  onSubmit={handleAboutSubmit} className="space-y-4">
              <input className='hidden' id="id" name="id" defaultValue={aboutData.id} />
              <div className="space-y-2">
                <Label htmlFor="about-title">Title</Label>
                <Input
                  id="about-title"
                  name="title"
                  defaultValue={aboutData.title || ''}
                  required
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="about-content">Content</Label>
                <Textarea
                  id="about-content"
                  name="content"
                  rows={6}
                  defaultValue={aboutData.content || ''}
                  required
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="about-image">Profile Image URL</Label>
                <Input
                  id="about-image"
                  name="profileImage"
                  defaultValue={aboutData.profile_image || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="about-meta">Meta Description</Label>
                <Textarea
                  id="about-meta"
                  name="metaDescription"
                  rows={2}
                  defaultValue={aboutData.meta_description || ''}
                />
              </div>
              
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Saving...' : 'Save About Page'}
              </Button>
            </form>
          ) : (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Title</Label>
                <p className="text-sm">{aboutData.title}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Content</Label>
                <p className="text-sm whitespace-pre-wrap">{aboutData.content}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Profile Image URL</Label>
                <p className="text-sm">{aboutData.profile_image}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Meta Description</Label>
                <p className="text-sm">{aboutData.meta_description}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="mb-8">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Contact Page</CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setEditingContact(!editingContact)}
          >
            {editingContact ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
          </Button>
        </CardHeader>
        <CardContent>
          {editingContact ? (
            <form onSubmit={handleContactSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="contact-title">Title</Label>
                <Input
                  id="contact-title"
                  name="title"
                  defaultValue={contactData.title || ''}
                  required
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-content">Content</Label>
                <Textarea
                  id="contact-content"
                  name="content"
                  rows={6}
                  defaultValue={contactData.content || ''}
                  required
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-email">Email Address</Label>
                <Input
                  id="contact-email"
                  name="emailAddress"
                  type="email"
                  defaultValue={contactData.email_address || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-social">Social Links (JSON)</Label>
                <Textarea
                  id="contact-social"
                  name="socialLinks"
                  rows={4}
                  defaultValue={contactData.social_links || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-meta">Meta Description</Label>
                <Textarea
                  id="contact-meta"
                  name="metaDescription"
                  rows={2}
                  defaultValue={contactData.meta_description || ''}
                />
              </div>
              
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Saving...' : 'Save Contact Page'}
              </Button>
            </form>
          ) : (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Title</Label>
                <p className="text-sm">{contactData.title}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Content</Label>
                <p className="text-sm whitespace-pre-wrap">{contactData.content}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Email Address</Label>
                <p className="text-sm">{contactData.email_address}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Social Links</Label>
                <p className="text-sm whitespace-pre-wrap">{contactData.social_links}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Meta Description</Label>
                <p className="text-sm">{contactData.meta_description}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  );
}
