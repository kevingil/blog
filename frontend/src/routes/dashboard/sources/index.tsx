import { useState, useEffect } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Globe, FileText, FileIcon, ExternalLink, Loader2, Trash2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination';
import { useToast } from '@/hooks/use-toast';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { getAllSources, updateSource, deleteSource } from '@/services/sources';
import type { ArticleSourceWithArticle, UpdateSourceRequest } from '@/services/types';

export const Route = createFileRoute('/dashboard/sources/')({
  component: SourcesPage,
});

function SourcesPage() {
  const [page, setPage] = useState(1);
  const [selectedSource, setSelectedSource] = useState<ArticleSourceWithArticle | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();

  // Form state for editing
  const [formData, setFormData] = useState({
    title: '',
    content: '',
    url: '',
    source_type: 'manual' as 'web' | 'manual' | 'pdf',
  });

  useEffect(() => {
    setPageTitle("Sources");
  }, [setPageTitle]);

  // Load form data when selecting a source
  useEffect(() => {
    if (selectedSource) {
      setFormData({
        title: selectedSource.title,
        content: selectedSource.content,
        url: selectedSource.url || '',
        source_type: selectedSource.source_type,
      });
    }
  }, [selectedSource]);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['all-sources', page],
    queryFn: () => getAllSources(page, 20),
  });

  const sources = data?.sources ?? [];
  const totalPages = data?.total_pages ?? 1;

  const handleEditSource = (source: ArticleSourceWithArticle) => {
    setSelectedSource(source);
    setIsEditing(true);
  };

  const handleCloseModal = () => {
    setSelectedSource(null);
    setIsEditing(false);
    setFormData({
      title: '',
      content: '',
      url: '',
      source_type: 'manual',
    });
  };

  const handleSave = async () => {
    if (!selectedSource) return;

    setIsSaving(true);
    try {
      const request: UpdateSourceRequest = {
        title: formData.title.trim(),
        content: formData.content.trim(),
        url: formData.url.trim() || undefined,
        source_type: formData.source_type,
      };

      await updateSource(selectedSource.id, request);
      toast({
        title: 'Success',
        description: 'Source updated successfully',
      });
      handleCloseModal();
      refetch();
    } catch (error) {
      console.error('Error updating source:', error);
      toast({
        title: 'Error',
        description: 'Failed to update source',
        variant: 'destructive',
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (sourceId: string) => {
    try {
      await deleteSource(sourceId);
      toast({
        title: 'Success',
        description: 'Source deleted successfully',
      });
      refetch();
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

  const truncateContent = (content: string, maxLength: number = 150) => {
    return content.length > maxLength ? content.substring(0, maxLength) + '...' : content;
  };

  const getSourceTypeIcon = (type: string) => {
    switch (type) {
      case 'web':
        return <Globe className="w-3 h-3" />;
      case 'pdf':
        return <FileIcon className="w-3 h-3" />;
      default:
        return <FileText className="w-3 h-3" />;
    }
  };

  const getSourceTypeBadgeVariant = (type: string) => {
    switch (type) {
      case 'web':
        return 'default';
      case 'pdf':
        return 'destructive';
      default:
        return 'secondary';
    }
  };

  if (error) return <div className="p-4">Error loading sources</div>;

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto">
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 animate-spin" />
          <span className="ml-2">Loading sources...</span>
        </div>
      ) : sources.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg font-medium mb-2">No sources found</p>
          <p className="text-sm">Sources are added to articles via the article editor.</p>
        </div>
      ) : (
        <>
          {/* Card Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {sources.map((source) => (
              <Card
                key={source.id}
                className="cursor-pointer transition-colors hover:border-primary/50"
                onClick={() => handleEditSource(source)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-sm font-medium line-clamp-2 flex-1">
                      {source.title || 'Untitled Source'}
                    </CardTitle>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-6 w-6 shrink-0"
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
                            onClick={() => handleDelete(source.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <Badge
                      variant={getSourceTypeBadgeVariant(source.source_type)}
                      className="text-xs"
                    >
                      {getSourceTypeIcon(source.source_type)}
                      <span className="ml-1 capitalize">{source.source_type}</span>
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {formatDate(source.created_at)}
                    </span>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <p className="text-sm text-muted-foreground line-clamp-3 mb-3">
                    {truncateContent(source.content)}
                  </p>
                  
                  {/* Article Link */}
                  <div className="border-t pt-2 mt-2">
                    <Link
                      to="/dashboard/blog/edit/$blogSlug"
                      params={{ blogSlug: source.article_slug }}
                      className="text-xs text-primary hover:underline flex items-center gap-1"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <FileText className="w-3 h-3" />
                      <span className="truncate">{source.article_title || 'Untitled Article'}</span>
                    </Link>
                  </div>

                  {/* URL if present */}
                  {source.url && (
                    <a
                      href={source.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 mt-2 truncate"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <ExternalLink className="w-3 h-3 flex-shrink-0" />
                      <span className="truncate">{source.url}</span>
                    </a>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-6 flex justify-center">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => setPage(Math.max(1, page - 1))}
                      className={page === 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    const pageNum = i + 1;
                    return (
                      <PaginationItem key={pageNum}>
                        <PaginationLink
                          onClick={() => setPage(pageNum)}
                          isActive={page === pageNum}
                          className="cursor-pointer"
                        >
                          {pageNum}
                        </PaginationLink>
                      </PaginationItem>
                    );
                  })}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => setPage(Math.min(totalPages, page + 1))}
                      className={page === totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </>
      )}

      {/* Edit Modal */}
      <Dialog open={isEditing} onOpenChange={(open) => !open && handleCloseModal()}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Source</DialogTitle>
            <DialogDescription>
              Update the source information below.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 mt-4">
            {/* Title and Source Type */}
            <div className="flex gap-4">
              <div className="flex-[2] space-y-2">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  placeholder="Source title"
                  value={formData.title}
                  onChange={(e) => setFormData((prev) => ({ ...prev, title: e.target.value }))}
                />
              </div>
              <div className="flex-1 space-y-2">
                <Label htmlFor="source-type">Source Type</Label>
                <Select
                  value={formData.source_type}
                  onValueChange={(value: 'web' | 'manual' | 'pdf') =>
                    setFormData((prev) => ({ ...prev, source_type: value }))
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

            {/* URL (for web sources) */}
            {formData.source_type === 'web' && (
              <div className="space-y-2">
                <Label htmlFor="url">URL</Label>
                <Input
                  id="url"
                  type="url"
                  placeholder="https://example.com/article"
                  value={formData.url}
                  onChange={(e) => setFormData((prev) => ({ ...prev, url: e.target.value }))}
                />
              </div>
            )}

            {/* Content */}
            <div className="space-y-2">
              <Label htmlFor="content">Content</Label>
              {formData.source_type === 'web' ? (
                <div className="min-h-[200px] max-h-[300px] p-3 border rounded-md bg-muted/50 overflow-y-auto">
                  <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                    {formData.content || 'No content available'}
                  </p>
                </div>
              ) : (
                <Textarea
                  id="content"
                  placeholder="Source content..."
                  className="min-h-[200px]"
                  value={formData.content}
                  onChange={(e) => setFormData((prev) => ({ ...prev, content: e.target.value }))}
                />
              )}
              {formData.source_type === 'web' && (
                <p className="text-xs text-muted-foreground">
                  Content was automatically extracted from the web source.
                </p>
              )}
            </div>

            {/* Article Info */}
            {selectedSource && (
              <div className="border-t pt-4">
                <p className="text-sm text-muted-foreground">
                  <span className="font-medium">Article:</span>{' '}
                  <Link
                    to="/dashboard/blog/edit/$blogSlug"
                    params={{ blogSlug: selectedSource.article_slug }}
                    className="text-primary hover:underline"
                  >
                    {selectedSource.article_title || 'Untitled Article'}
                  </Link>
                </p>
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={handleCloseModal}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={isSaving}>
                {isSaving ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  'Save Changes'
                )}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  );
}
