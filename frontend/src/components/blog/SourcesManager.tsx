import { useState, useEffect, useRef } from 'react';
import { Plus, ExternalLink, Edit2, Trash2, Globe, FileText, Loader2, FileIcon, X } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
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

interface SourcesManagerProps {
  articleId: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

interface SourceFormData {
  title: string;
  content: string;
  url: string;
  source_type: 'web' | 'manual' | 'pdf';
}

export function SourcesManager({ articleId, isOpen, onOpenChange }: SourcesManagerProps) {
  const { toast } = useToast();
  const [sources, setSources] = useState<ArticleSource[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedSource, setSelectedSource] = useState<ArticleSource | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
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
  
  const titleInputRef = useRef<HTMLInputElement>(null);

  // Load sources when modal opens
  useEffect(() => {
    if (isOpen && articleId) {
      loadSources();
    }
  }, [isOpen, articleId]);

  // Reset form when modal opens/closes or selected source changes
  useEffect(() => {
    if (selectedSource && isEditing) {
      setFormData({
        title: selectedSource.title,
        content: selectedSource.content,
        url: selectedSource.url || '',
        source_type: selectedSource.source_type,
      });
      setUseUrlScraping(selectedSource.source_type === 'web');
    } else if (isCreating) {
      setFormData({
        title: '',
        content: '',
        url: '',
        source_type: 'manual'
      });
      setUseUrlScraping(false);
    }
  }, [selectedSource, isEditing, isCreating]);

  // Auto-focus title input when editing/creating
  useEffect(() => {
    if ((isEditing || isCreating) && titleInputRef.current) {
      setTimeout(() => {
        titleInputRef.current?.focus();
      }, 100);
    }
  }, [isEditing, isCreating]);

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

  const handleCreateNew = () => {
    setSelectedSource(null);
    setIsEditing(false);
    setIsCreating(true);
  };

  const handleEditSource = (source: ArticleSource) => {
    setSelectedSource(source);
    setIsCreating(false);
    setIsEditing(true);
  };

  const handleCancelEdit = () => {
    setSelectedSource(null);
    setIsEditing(false);
    setIsCreating(false);
    setFormData({
      title: '',
      content: '',
      url: '',
      source_type: 'manual'
    });
    setUseUrlScraping(false);
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    
    // Validation logic
    const isWebSource = isEditing ? formData.source_type === 'web' : useUrlScraping;
    const requiresContent = isEditing ? formData.source_type !== 'web' : !useUrlScraping;
    
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
      if (isEditing && selectedSource) {
        // Update existing source
        const request: UpdateSourceRequest = {
          title: formData.title.trim(),
          content: formData.content.trim(),
          url: formData.url.trim() || undefined,
          source_type: formData.source_type,
        };

        const updatedSource = await updateSource(selectedSource.id, request);
        setSources(prev => prev.map(s => s.id === updatedSource.id ? updatedSource : s));
        setSelectedSource(updatedSource);
        toast({
          title: 'Success',
          description: 'Source updated successfully',
        });
      } else if (isCreating) {
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

        setSources(prev => [newSource, ...prev]);
        setSelectedSource(newSource);
        setIsCreating(false);
        setIsEditing(true);
        toast({
          title: 'Success',
          description: useUrlScraping ? 'Source scraped and added successfully' : 'Source added successfully',
        });
      }
    } catch (error) {
      console.error('Error saving source:', error);
      toast({
        title: 'Error',
        description: isEditing ? 'Failed to update source' : 'Failed to add source',
        variant: 'destructive',
      });
    } finally {
      setIsAddingSource(false);
      setIsScrapingUrl(false);
    }
  };

  const handleDeleteSource = async (sourceId: string) => {
    try {
      await deleteSource(sourceId);
      setSources(prev => prev.filter(s => s.id !== sourceId));
      if (selectedSource?.id === sourceId) {
        handleCancelEdit();
      }
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
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-[90vw] w-[90vw] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FileText className="w-5 h-5" />
            Sources & References
          </DialogTitle>
          <DialogDescription>
            Manage sources and references for this article. Add web links to scrape content automatically, or add manual sources.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 flex gap-6 min-h-0">
          {/* Left Panel - Sources List */}
          <div className="w-1/3 flex flex-col">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium">Sources ({sources.length})</h3>
              <Button onClick={handleCreateNew} size="sm">
                <Plus className="w-4 h-4 mr-2" />
                Add Source
              </Button>
            </div>

            <div className="flex-1 overflow-y-auto space-y-3">
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
                sources.map((source) => (
                  <Card 
                    key={source.id} 
                    className={`border cursor-pointer transition-colors ${
                      selectedSource?.id === source.id ? 'border-blue-500 bg-blue-50 dark:bg-blue-950' : 'hover:bg-gray-50 dark:hover:bg-gray-800'
                    }`}
                    onClick={() => handleEditSource(source)}
                  >
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
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button 
                                variant="ghost" 
                                size="sm"
                                onClick={(e) => e.stopPropagation()}
                              >
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
                            onClick={(e) => e.stopPropagation()}
                          >
                            <ExternalLink className="w-3 h-3 flex-shrink-0" />
                            <span className="truncate">{source.url}</span>
                          </a>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </div>

          {/* Right Panel - Edit Form */}
          <div className="w-full flex flex-col">
            {(isEditing || isCreating) ? (
              <>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium">
                    {isCreating ? 'Add New Source' : 'Edit Source'}
                  </h3>
                  <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
                    <X className="w-4 h-4" />
                  </Button>
                </div>

                <form onSubmit={handleSubmit} className="flex-1 flex flex-col space-y-4">
                  {/* URL Scraping Toggle */}
                  {isCreating && (
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
                  {isEditing ? (
                    <div className="flex gap-4">
                      {/* Title Input */}
                      <div className="flex-[2] space-y-2">
                        <Label htmlFor="title">Title</Label>
                        <Input
                          ref={titleInputRef}
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
                        ref={titleInputRef}
                        id="title"
                        placeholder="Source title"
                        value={formData.title}
                        onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                      />
                    </div>
                  )}

                  {/* URL Input (when scraping or editing web source) */}
                  {(useUrlScraping || (isEditing && formData.source_type === 'web')) && (
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
                  {(!useUrlScraping && !isEditing) || (isEditing && formData.source_type !== 'web') ? (
                    <div className="space-y-2 flex-1 flex flex-col">
                      <Label htmlFor="content">Content</Label>
                      <Textarea
                        id="content"
                        placeholder={
                          isEditing && formData.source_type === 'pdf' 
                            ? "PDF content or summary..." 
                            : "Source content or summary..."
                        }
                        className="flex-1 min-h-[300px] max-h-[calc(100vh-460px)]"
                        value={formData.content}
                        onChange={(e) => setFormData(prev => ({ ...prev, content: e.target.value }))}
                      />
                      {isEditing && formData.source_type === 'pdf' && (
                        <p className="text-xs text-muted-foreground">
                          PDF file content was extracted at upload time. You can edit this content as needed.
                        </p>
                      )}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      {isEditing && formData.source_type === 'web' 
                        ? "Content is managed automatically for web sources."
                        : "Content will be automatically extracted from the URL."
                      }
                    </p>
                  )}

                  <div className="flex gap-2 pt-4 ml-auto">
                    <Button variant="outline" onClick={handleCancelEdit} type="button">
                      Cancel
                    </Button>
                    <Button 
                      type="submit"
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
                          {isEditing ? 'Updating...' : 'Adding...'}
                        </>
                      ) : (
                        isEditing ? 'Update Source' : 'Add Source'
                      )}
                    </Button>
                  </div>
                </form>
              </>
            ) : (
              <div className="flex-1 flex items-center justify-center text-center text-muted-foreground">
                <div className="h-[300px]">
                  <FileText className="w-16 h-16 mx-auto mb-4 opacity-30" />
                  <p className="text-lg font-medium mb-2">Select a source to edit</p>
                  <p className="text-sm">Choose a source from the list or create a new one</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
