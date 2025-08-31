import { useState, useEffect } from 'react';
import { Plus, ExternalLink, Edit2, Trash2, Globe, FileText, Loader2, FileIcon } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { 
  Drawer, 
  DrawerContent, 
  DrawerHeader, 
  DrawerTitle, 
  DrawerDescription,
  DrawerFooter,
  DrawerClose
} from '@/components/ui/drawer';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { 
  getArticleSources, 
  createSource, 
  scrapeAndCreateSource, 
  updateSource, 
  deleteSource 
} from '@/services/sources';
import type { ArticleSource, CreateSourceRequest, UpdateSourceRequest } from '@/services/types';

interface SourcesDrawerProps {
  articleId: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

interface AddSourceModalProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  articleId: string;
  editingSource: ArticleSource | null;
  onSourceAdded: (source: ArticleSource) => void;
  onSourceUpdated: (source: ArticleSource) => void;
}

interface SourceFormData {
  title: string;
  content: string;
  url: string;
  source_type: 'web' | 'manual' | 'pdf';
}

function AddSourceModal({ 
  isOpen, 
  onOpenChange, 
  articleId, 
  editingSource, 
  onSourceAdded, 
  onSourceUpdated 
}: AddSourceModalProps) {
  const { toast } = useToast();
  const [isAddingSource, setIsAddingSource] = useState(false);
  const [isScrapingUrl, setIsScrapingUrl] = useState(false);
  
  // Form state
  const [formData, setFormData] = useState<SourceFormData>({
    title: '',
    content: '',
    url: '',
    source_type: 'manual'
  });
  const [useUrlScraping, setUseUrlScraping] = useState(false);

  // Reset form when modal opens/closes or editing source changes
  useEffect(() => {
    if (editingSource) {
      setFormData({
        title: editingSource.title,
        content: editingSource.content,
        url: editingSource.url || '',
        source_type: editingSource.source_type,
      });
      setUseUrlScraping(editingSource.source_type === 'web');
    } else {
      setFormData({
        title: '',
        content: '',
        url: '',
        source_type: 'manual'
      });
      setUseUrlScraping(false);
    }
  }, [editingSource, isOpen]);

  const handleModalClose = (open: boolean) => {
    if (!open) {
      // Reset loading states when closing
      setIsAddingSource(false);
      setIsScrapingUrl(false);
    }
    onOpenChange(open);
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    
    // Validation logic
    const isWebSource = editingSource ? formData.source_type === 'web' : useUrlScraping;
    const requiresContent = editingSource ? formData.source_type !== 'web' : !useUrlScraping;
    
    if (!formData.title.trim()) {
      toast({
        title: 'Error',
        description: 'Title is required',
        variant: 'destructive',
      });
      return;
    }

    if (requiresContent && !formData.content.trim()) {
      toast({
        title: 'Error',
        description: 'Content is required',
        variant: 'destructive',
      });
      return;
    }

    if (isWebSource && !formData.url.trim()) {
      toast({
        title: 'Error',
        description: 'URL is required for web sources',
        variant: 'destructive',
      });
      return;
    }

    setIsAddingSource(true);
    try {
      if (editingSource) {
        // Update existing source
        const request: UpdateSourceRequest = {
          title: formData.title.trim(),
          content: formData.content.trim(),
          url: formData.url.trim() || undefined,
          source_type: formData.source_type,
        };

        const updatedSource = await updateSource(editingSource.id, request);
        onSourceUpdated(updatedSource);
        toast({
          title: 'Success',
          description: 'Source updated successfully',
        });
      } else {
        // Create new source
        let newSource: ArticleSource;

        if (useUrlScraping && formData.url.trim()) {
          setIsScrapingUrl(true);
          newSource = await scrapeAndCreateSource({
            article_id: articleId,
            url: formData.url.trim(),
          });
        } else {
          const request: CreateSourceRequest = {
            article_id: articleId,
            title: formData.title.trim(),
            content: formData.content.trim(),
            url: formData.url.trim() || undefined,
            source_type: useUrlScraping ? 'web' : 'manual',
          };
          newSource = await createSource(request);
        }

        onSourceAdded(newSource);
        toast({
          title: 'Success',
          description: useUrlScraping ? 'Source scraped and added successfully' : 'Source added successfully',
        });
      }

      // Use setTimeout to ensure state updates complete before closing
      setTimeout(() => {
        handleModalClose(false);
      }, 100);
    } catch (error) {
      console.error('Error saving source:', error);
      toast({
        title: 'Error',
        description: editingSource ? 'Failed to update source' : 'Failed to add source',
        variant: 'destructive',
      });
    } finally {
      setIsAddingSource(false);
      setIsScrapingUrl(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleModalClose}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {editingSource ? 'Edit Source' : 'Add New Source'}
          </DialogTitle>
          <DialogDescription>
            {editingSource 
              ? 'Update the source information below.' 
              : 'Add a new source or reference to your article.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* URL Scraping Toggle */}
          {!editingSource && (
            <div className="flex items-center justify-between">
              <Label htmlFor="use-url-scraping" className="text-sm font-medium">
                Scrape from URL
              </Label>
              <Switch
                id="use-url-scraping"
                checked={useUrlScraping}
                onCheckedChange={setUseUrlScraping}
              />
            </div>
          )}

          {/* Source Type and Title Row (when editing) */}
          {editingSource ? (
            <div className="flex gap-4">
              {/* Title Input */}
              <div className="flex-[2] space-y-2">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  placeholder="Source title"
                  value={formData.title}
                  onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                />
              </div>

              {/* Source Type Dropdown */}
              <div className="flex-1 space-y-2">
                <Label htmlFor="source-type">Source Type</Label>
                <Select
                  value={formData.source_type}
                  onValueChange={(value: 'web' | 'manual' | 'pdf') => 
                    setFormData(prev => ({ ...prev, source_type: value }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select source type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4" />
                        Manual
                      </div>
                    </SelectItem>
                    <SelectItem value="web">
                      <div className="flex items-center gap-2">
                        <Globe className="w-4 h-4" />
                        Web
                      </div>
                    </SelectItem>
                    <SelectItem value="pdf">
                      <div className="flex items-center gap-2">
                        <FileIcon className="w-4 h-4" />
                        PDF
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          ) : (
            /* Title Input for new sources */
            <div className="space-y-2">
              <Label htmlFor="title">Title</Label>
              <Input
                id="title"
                placeholder="Source title"
                value={formData.title}
                onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
              />
            </div>
          )}

          {/* URL Input (when scraping or editing web source) */}
          {(useUrlScraping || (editingSource && formData.source_type === 'web')) && (
            <div className="space-y-2">
              <Label htmlFor="url">URL</Label>
              <Input
                id="url"
                type="url"
                placeholder="https://example.com/article"
                value={formData.url}
                onChange={(e) => setFormData(prev => ({ ...prev, url: e.target.value }))}
              />
            </div>
          )}

          {/* Content Input (only when not using URL scraping or editing non-web sources) */}
          {(!useUrlScraping && !editingSource) || (editingSource && formData.source_type !== 'web') ? (
            <div className="space-y-2">
              <Label htmlFor="content">Content</Label>
              <Textarea
                id="content"
                placeholder={
                  editingSource && formData.source_type === 'pdf' 
                    ? "PDF content or summary..." 
                    : "Source content or summary..."
                }
                rows={6}
                className="resize-none min-h-[150px] max-h-[300px] overflow-y-auto"
                value={formData.content}
                onChange={(e) => setFormData(prev => ({ ...prev, content: e.target.value }))}
              />
              {editingSource && formData.source_type === 'pdf' && (
                <p className="text-xs text-muted-foreground">
                  PDF file content was extracted at upload time. You can edit this content as needed.
                </p>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              {editingSource && formData.source_type === 'web' 
                ? "Content is managed automatically for web sources."
                : "Content will be automatically extracted from the URL."
              }
            </p>
          )}
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button 
            onClick={handleSubmit}
            disabled={isAddingSource || isScrapingUrl}
          >
            {isScrapingUrl ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Scraping...
              </>
            ) : isAddingSource ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                {editingSource ? 'Updating...' : 'Adding...'}
              </>
            ) : (
              editingSource ? 'Update Source' : 'Add Source'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function SourcesDrawer({ articleId, isOpen, onOpenChange }: SourcesDrawerProps) {
  const { toast } = useToast();
  const [sources, setSources] = useState<ArticleSource[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [editingSource, setEditingSource] = useState<ArticleSource | null>(null);
  const [addSourceModalOpen, setAddSourceModalOpen] = useState(false);

  // Load sources when drawer opens
  useEffect(() => {
    if (isOpen && articleId) {
      loadSources();
    }
  }, [isOpen, articleId]);

  const loadSources = async () => {
    setIsLoading(true);
    try {
      const sourcesData = await getArticleSources(articleId);
      setSources(sourcesData);
    } catch (error) {
      console.error('Error loading sources:', error);
      toast({
        title: 'Error',
        description: 'Failed to load sources',
        variant: 'destructive',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleSourceAdded = (source: ArticleSource) => {
    setSources(prev => [source, ...prev]);
  };

  const handleSourceUpdated = (updatedSource: ArticleSource) => {
    setSources(prev => prev.map(s => s.id === updatedSource.id ? updatedSource : s));
    setEditingSource(null);
  };

  const handleEditSource = (source: ArticleSource) => {
    setEditingSource(source);
    setAddSourceModalOpen(true);
  };

  const handleDeleteSource = async (sourceId: string) => {
    try {
      await deleteSource(sourceId);
      setSources(prev => prev.filter(s => s.id !== sourceId));
      toast({
        title: 'Success',
        description: 'Source deleted successfully',
      });
    } catch (error) {
      console.error('Error deleting source:', error);
      toast({
        title: 'Error',
        description: 'Failed to delete source',
        variant: 'destructive',
      });
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const truncateContent = (content: string, maxLength: number = 100) => {
    return content.length > maxLength ? content.substring(0, maxLength) + '...' : content;
  };

  return (
    <Drawer open={isOpen} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="w-full sm:max-w-2xl ml-auto h-full">
        <DrawerHeader className="border-b">
          <DrawerTitle className="flex items-center gap-2">
            <FileText className="w-5 h-5" />
            Sources & References
          </DrawerTitle>
          <DrawerDescription>
            Manage sources and references for this article. Add web links to scrape content automatically, or add manual sources.
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {/* Add Source Button */}
          <Button 
            className="w-full" 
            onClick={() => {
              setEditingSource(null);
              setAddSourceModalOpen(true);
            }}
          >
            <Plus className="w-4 h-4 mr-2" />
            Add Source
          </Button>

          {/* Sources List */}
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="w-6 h-6 animate-spin" />
              <span className="ml-2">Loading sources...</span>
            </div>
          ) : sources.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>No sources added yet.</p>
              <p className="text-sm">Add your first source to get started.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {sources.map((source) => (
                <Card key={source.id} className="border">
                  <CardHeader className="pb-2">
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <CardTitle className="text-sm font-medium line-clamp-2">
                          {source.title}
                        </CardTitle>
                        <div className="flex items-center gap-2 mt-1">
                          <Badge variant={source.source_type === 'web' ? 'default' : source.source_type === 'pdf' ? 'destructive' : 'secondary'} className="text-xs">
                            {source.source_type === 'web' ? (
                              <>
                                <Globe className="w-3 h-3 mr-1" />
                                Web
                              </>
                            ) : source.source_type === 'pdf' ? (
                              <>
                                <FileIcon className="w-3 h-3 mr-1" />
                                PDF
                              </>
                            ) : (
                              <>
                                <FileText className="w-3 h-3 mr-1" />
                                Manual
                              </>
                            )}
                          </Badge>
                          <span className="text-xs text-muted-foreground">
                            {formatDate(source.created_at)}
                          </span>
                        </div>
                      </div>
                      <div className="flex items-center gap-1 ml-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditSource(source)}
                        >
                          <Edit2 className="w-3 h-3" />
                        </Button>
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <Trash2 className="w-3 h-3" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete Source</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to delete this source? This action cannot be undone.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => handleDeleteSource(source.id)}
                                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                              >
                                Delete
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <p className="text-sm text-muted-foreground line-clamp-3">
                      {truncateContent(source.content)}
                    </p>
                    {source.url && (
                      <div className="mt-2">
                        <a
                          href={source.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 truncate"
                        >
                          <ExternalLink className="w-3 h-3 flex-shrink-0" />
                          <span className="truncate">{source.url}</span>
                        </a>
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>

        <DrawerFooter className="border-t">
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>{sources.length} source{sources.length !== 1 ? 's' : ''}</span>
            <DrawerClose asChild>
              <Button variant="outline">Close</Button>
            </DrawerClose>
          </div>
        </DrawerFooter>
      </DrawerContent>
      
      {/* Add Source Modal - Outside of drawer to avoid z-index conflicts */}
      <AddSourceModal
        isOpen={addSourceModalOpen}
        onOpenChange={setAddSourceModalOpen}
        articleId={articleId}
        editingSource={editingSource}
        onSourceAdded={handleSourceAdded}
        onSourceUpdated={handleSourceUpdated}
      />
    </Drawer>
  );
}
