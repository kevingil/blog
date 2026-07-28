import { useState, useEffect } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useToast } from '@/hooks/use-toast';
import { useAuth } from '@/services/auth/auth';
import { generateArticle } from '@/services/llm/articles';
import { scrapeAndCreateSource } from '@/services/sources';
import { AIChatLanding, AttachedSource } from '@/components/chat/AIChatLanding';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/')({
  component: DashboardIndex,
});

function DashboardIndex() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [isGenerating, setIsGenerating] = useState(false);
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("AI Copilot");
  }, [setPageTitle]);

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
      // Step 1: Create draft shell + start chat session
      const { article, request_id } = await generateArticle(
        prompt,
        undefined,
      );

      // Step 2: Attach sources if any (in parallel)
      if (sources.length > 0) {
        const sourcePromises = sources.map(source =>
          scrapeAndCreateSource({
            article_id: article.id.toString(),
            url: source.url
          }).catch(err => {
            console.error(`Failed to scrape source ${source.url}:`, err);
            return null;
          })
        );

        await Promise.all(sourcePromises);
      }

      toast({
        title: "Generating",
        description: sources.length > 0
          ? "Started generation with sources attached"
          : "Started article generation",
      });

      // Step 3: Navigate to editor with the active session
      navigate({
        to: `/dashboard/blog/edit/${article.slug}`,
        search: { requestId: request_id },
      });
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
