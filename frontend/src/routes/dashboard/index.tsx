import { useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useToast } from '@/hooks/use-toast';
import { useAuth } from '@/services/auth/auth';
import { generateArticle } from '@/services/llm/articles';
import { scrapeAndCreateSource } from '@/services/sources';
import { Article } from '@/services/types';
import { AIChatLanding, AttachedSource } from '@/components/chat/AIChatLanding';

export const Route = createFileRoute('/dashboard/')({
  component: DashboardIndex,
});

function DashboardIndex() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [isGenerating, setIsGenerating] = useState(false);

  const handleGenerate = async (prompt: string, sources: AttachedSource[]) => {
    if (!user?.id) {
      toast({
        title: "Error",
        description: "User not found. Please log in again.",
        variant: "destructive",
      });
      return;
    }

    setIsGenerating(true);
  
    try {
      // Step 1: Generate the article
      const newGeneratedArticle: Article = await generateArticle(
        prompt, 
        "Untitled Article", 
        user.id
      );
      
      // Step 2: Attach sources if any (in parallel)
      if (sources.length > 0) {
        const sourcePromises = sources.map(source => 
          scrapeAndCreateSource({
            article_id: newGeneratedArticle.id.toString(),
            url: source.url
          }).catch(err => {
            console.error(`Failed to scrape source ${source.url}:`, err);
            return null; // Don't fail the whole process if one source fails
          })
        );
        
        await Promise.all(sourcePromises);
      }
      
      toast({
        title: "Success",
        description: sources.length > 0 
          ? "Article generated with sources attached"
          : "Article generated successfully",
      });
      
      // Step 3: Navigate to editor
      navigate({ to: `/dashboard/blog/edit/${newGeneratedArticle.slug}` });
    } catch (err) {
      console.error("Generation failed:", err);
      toast({
        title: "Error",
        description: "Failed to generate article. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <section className="flex-1">
      <AIChatLanding 
        onGenerate={handleGenerate}
        isGenerating={isGenerating}
      />
    </section>
  );
}
