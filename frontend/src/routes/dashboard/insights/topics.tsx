import { useState, useEffect } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Tag, Plus, Pencil, Trash2, Loader2, Hash } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
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
import { useToast } from '@/hooks/use-toast';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import { 
  listTopics, 
  createTopic, 
  updateTopic, 
  deleteTopic,
  type InsightTopic,
  type CreateTopicRequest,
  type UpdateTopicRequest,
} from '@/services/insights';

export const Route = createFileRoute('/dashboard/insights/topics')({
  component: TopicsPage,
});

function TopicsPage() {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTopic, setEditingTopic] = useState<InsightTopic | null>(null);
  const [formData, setFormData] = useState<CreateTopicRequest>({
    name: '',
    description: '',
    keywords: [],
  });
  const [keywordsInput, setKeywordsInput] = useState('');
  const { toast } = useToast();
  const { setPageTitle } = useAdminDashboard();
  const queryClient = useQueryClient();

  useEffect(() => {
    setPageTitle("Topics");
  }, [setPageTitle]);

  const { data: topics = [], isLoading } = useQuery({
    queryKey: ['insight-topics'],
    queryFn: listTopics,
  });

  const createMutation = useMutation({
    mutationFn: createTopic,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insight-topics'] });
      toast({ title: 'Success', description: 'Topic created successfully' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to create topic', variant: 'destructive' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTopicRequest }) => updateTopic(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insight-topics'] });
      toast({ title: 'Success', description: 'Topic updated successfully' });
      handleCloseDialog();
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to update topic', variant: 'destructive' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTopic,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['insight-topics'] });
      toast({ title: 'Success', description: 'Topic deleted successfully' });
    },
    onError: () => {
      toast({ title: 'Error', description: 'Failed to delete topic', variant: 'destructive' });
    },
  });

  const handleOpenCreate = () => {
    setEditingTopic(null);
    setFormData({ name: '', description: '', keywords: [] });
    setKeywordsInput('');
    setIsDialogOpen(true);
  };

  const handleOpenEdit = (topic: InsightTopic) => {
    setEditingTopic(topic);
    setFormData({
      name: topic.name,
      description: topic.description || '',
      keywords: topic.keywords || [],
    });
    setKeywordsInput(topic.keywords?.join(', ') || '');
    setIsDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setEditingTopic(null);
    setFormData({ name: '', description: '', keywords: [] });
    setKeywordsInput('');
  };

  const handleSubmit = () => {
    const keywords = keywordsInput
      .split(',')
      .map((k) => k.trim())
      .filter((k) => k.length > 0);

    const data = {
      ...formData,
      keywords,
    };

    if (editingTopic) {
      updateMutation.mutate({ id: editingTopic.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  return (
    <section className="flex-1 p-0 md:p-4 overflow-auto">
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-lg font-semibold">Insight Topics</h2>
          <p className="text-sm text-muted-foreground">
            Topics help organize and categorize your insights from crawled content.
          </p>
        </div>
        <Button onClick={handleOpenCreate}>
          <Plus className="w-4 h-4 mr-2" />
          New Topic
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 animate-spin" />
          <span className="ml-2">Loading topics...</span>
        </div>
      ) : topics.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <Tag className="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg font-medium mb-2">No topics yet</p>
          <p className="text-sm mb-4">Create topics to organize your insights by theme or category.</p>
          <Button onClick={handleOpenCreate}>
            <Plus className="w-4 h-4 mr-2" />
            Create Your First Topic
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {topics.map((topic) => (
            <Card key={topic.id}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <Tag className="w-4 h-4 text-primary" />
                    <CardTitle className="text-base">{topic.name}</CardTitle>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => handleOpenEdit(topic)}
                    >
                      <Pencil className="w-3 h-3" />
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-7 w-7">
                          <Trash2 className="w-3 h-3" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete Topic</AlertDialogTitle>
                          <AlertDialogDescription>
                            Are you sure you want to delete "{topic.name}"? This will not delete associated insights.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => deleteMutation.mutate(topic.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
                {topic.description && (
                  <CardDescription className="line-clamp-2">
                    {topic.description}
                  </CardDescription>
                )}
              </CardHeader>
              <CardContent>
                {topic.keywords && topic.keywords.length > 0 && (
                  <div className="flex flex-wrap gap-1 mb-3">
                    {topic.keywords.slice(0, 5).map((keyword, i) => (
                      <Badge key={i} variant="outline" className="text-xs">
                        <Hash className="w-2 h-2 mr-1" />
                        {keyword}
                      </Badge>
                    ))}
                    {topic.keywords.length > 5 && (
                      <Badge variant="outline" className="text-xs">
                        +{topic.keywords.length - 5}
                      </Badge>
                    )}
                  </div>
                )}
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{topic.content_count} content items</span>
                  <span>Created {formatDate(topic.created_at)}</span>
                </div>
                {topic.is_auto_generated && (
                  <Badge variant="secondary" className="mt-2 text-xs">
                    Auto-generated
                  </Badge>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={(open) => !open && handleCloseDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingTopic ? 'Edit Topic' : 'Create Topic'}</DialogTitle>
            <DialogDescription>
              {editingTopic 
                ? 'Update the topic details below.'
                : 'Create a new topic to organize your insights.'}
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="e.g., AI/ML, Web Development, DevOps"
                value={formData.name}
                onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
              />
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                placeholder="Describe what content this topic covers..."
                value={formData.description || ''}
                onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
              />
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="keywords">Keywords</Label>
              <Input
                id="keywords"
                placeholder="machine learning, neural networks, deep learning"
                value={keywordsInput}
                onChange={(e) => setKeywordsInput(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Comma-separated keywords to help match content to this topic.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={handleCloseDialog}>
              Cancel
            </Button>
            <Button 
              onClick={handleSubmit}
              disabled={!formData.name.trim() || createMutation.isPending || updateMutation.isPending}
            >
              {(createMutation.isPending || updateMutation.isPending) ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving...
                </>
              ) : editingTopic ? (
                'Update Topic'
              ) : (
                'Create Topic'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
