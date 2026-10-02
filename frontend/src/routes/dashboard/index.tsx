import { useState, useEffect, useRef } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useToast } from '@/hooks/use-toast';
import { useAuth } from '@/services/auth/auth';
import { generateArticle } from '@/services/llm/articles';
import { scrapeAndCreateSource } from '@/services/sources';
import { AIChatLanding, AttachedSource } from '@/components/chat/AIChatLanding';
import { useAdminDashboard } from '@/services/dashboard/dashboard';
import ArticleEditor from '@/components/blog/Editor';

export const Route = createFileRoute('/dashboard/')({
  component: DashboardIndex,
});

function DashboardIndex() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [isGenerating, setIsGenerating] = useState(false);
  const [session, setSession] = useState(0);
  const sessionRef = useRef(session);
  sessionRef.current = session;
  const { setPageTitle, blankEditor, setBlankEditor } = useAdminDashboard();

  useEffect(() => {
    const reset = () => {
      setSession((value) => value + 1);
      setIsGenerating(false);
      setBlankEditor(false);
    };
    window.addEventListener("writing-session-reset", reset);
    return () => window.removeEventListener("writing-session-reset", reset);
  }, [setBlankEditor]);

  useEffect(() => {
    setPageTitle("New");
  }, [setPageTitle]);

  const handleGenerate = async (prompt: string, sources: AttachedSource[]) => {
    const generation = sessionRef.current;
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

      if (sessionRef.current !== generation) return;

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
      if (sessionRef.current !== generation) return;
      console.error("Generation failed:", err);
      toast({
        title: "Error",
        description: "Failed to generate article. Please try again.",
        variant: "destructive",
      });
    } finally {
      if (sessionRef.current === generation) {
        setIsGenerating(false);
      }
    }
  };

  if (blankEditor) {
    return <ArticleEditor isNew />;
  }

  return (
    <section className="flex-1">
      <AIChatLanding
        key={session}
        onGenerate={handleGenerate}
        isGenerating={isGenerating}
      />
    </section>
  );
}
