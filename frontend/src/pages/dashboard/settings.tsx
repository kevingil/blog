'use client';

import { useState, useEffect } from 'react';
import { getAboutPage, getContactPage, updateAboutPage, updateContactPage } from '@/db/queries';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { SettingsSkeleton } from './settingsLoading';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
import { Pencil, X } from 'lucide-react';

interface AboutPage {
  id: number;
  title: string | null;
  content: string | null;
  profileImage?: string | null;
  metaDescription?: string | null;
  lastUpdated: string | null;
}

interface ContactPage {
  id: number;
  title: string | null;
  content: string | null;
  emailAddress: string | null;
  socialLinks?: string | null;
  metaDescription?: string | null;
  lastUpdated: string | null;
}

export function Settings() {
  const [aboutData, setAboutData] = useState<AboutPage | null>(null);
  const [contactData, setContactData] = useState<ContactPage | null>(null);
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
      setAboutData(about);
      setContactData(contact);
    };
    loadData();
  }, []);

  async function handleAboutSubmit(formData: FormData) {
    setIsLoading(true);
    try {
      if (aboutData) {
        const data = {
          id: aboutData.id,
          title: formData.get('title') as string,
          content: formData.get('content') as string,
          profileImage: formData.get('profileImage') as string,
          metaDescription: formData.get('metaDescription') as string,
          lastUpdated: new Date().toISOString(),
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

  async function handleContactSubmit(formData: FormData) {
    setIsLoading(true);
    try {
      if (contactData) {
        const data = {
          id: contactData.id,
          title: formData.get('title') as string,
          content: formData.get('content') as string,
          emailAddress: formData.get('emailAddress') as string,
          socialLinks: formData.get('socialLinks') as string,
        metaDescription: formData.get('metaDescription') as string,
        lastUpdated: new Date().toISOString(),
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
            <form action={handleAboutSubmit} className="space-y-4">
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
                  defaultValue={aboutData.profileImage || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="about-meta">Meta Description</Label>
                <Textarea
                  id="about-meta"
                  name="metaDescription"
                  rows={2}
                  defaultValue={aboutData.metaDescription || ''}
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
                <p className="text-sm">{aboutData.profileImage}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Meta Description</Label>
                <p className="text-sm">{aboutData.metaDescription}</p>
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
            <form action={handleContactSubmit} className="space-y-4">
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
                  defaultValue={contactData.emailAddress || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-social">Social Links (JSON)</Label>
                <Textarea
                  id="contact-social"
                  name="socialLinks"
                  rows={4}
                  defaultValue={contactData.socialLinks || ''}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="contact-meta">Meta Description</Label>
                <Textarea
                  id="contact-meta"
                  name="metaDescription"
                  rows={2}
                  defaultValue={contactData.metaDescription || ''}
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
                <p className="text-sm">{contactData.emailAddress}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Social Links</Label>
                <p className="text-sm whitespace-pre-wrap">{contactData.socialLinks}</p>
              </div>
              
              <div className="space-y-2">
                <Label>Meta Description</Label>
                <p className="text-sm">{contactData.metaDescription}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </section>
  );
}
