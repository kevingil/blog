import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer"
import { useAuth } from '@/services/auth/auth';
import { generateArticle } from '@/services/llm/articles';
import { useNavigate } from '@tanstack/react-router';

interface GenerateArticleDrawerProps {
  children: React.ReactNode;
}

export function GenerateArticleDrawer({ children }: GenerateArticleDrawerProps) {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [isGenerating, setIsGenerating] = useState(false);
  const [aiArticleTitle, setAiArticleTitle] = useState<string>('');
  const [aiArticlePrompt, setAiArticlePrompt] = useState<string>('');
  const [formError, setFormError] = useState<string>('');

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();

    const title = aiArticleTitle.trim();
    const prompt = aiArticlePrompt.trim();
    if (!title && !prompt) {
      setFormError('Add a title, a prompt, or both.');
      return;
    }

    setIsGenerating(true);
    setFormError('');

    try {
      if (!user?.id) {
        throw new Error("User not found");
      }
      const { article, request_id } = await generateArticle(prompt, title);
      navigate({
        to: `/dashboard/blog/edit/${article.slug}`,
        search: { requestId: request_id },
      });
    } catch (err) {
      console.error("Generation failed:", err);
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <Drawer>
      <DrawerTrigger asChild>
        {children}
      </DrawerTrigger>
      <DrawerContent className="w-full max-w-3xl mx-auto">
        <DrawerHeader>
          <DrawerTitle>Generate Article</DrawerTitle>
        </DrawerHeader>

        <form onSubmit={handleGenerate} className="space-y-4 px-4 pb-4">
          <div>
            <p className="pb-4 text-muted-foreground text-[0.9rem]">Start with a title, a freeform prompt, or both. Title-only generation will draft the article from that topic.</p>
            <label htmlFor="title" className="block font-bold text-gray-500 text-sm mb-2">
              Title
            </label>
            <Input
              id="title"
              type="text"
              placeholder="Hybrid Human-Agent Engineering Teams"
              value={aiArticleTitle}
              onChange={(e) => setAiArticleTitle(e.target.value)}
            />
          </div>

          <div>
            <label htmlFor="prompt" className="block font-bold text-gray-500 text-sm mb-2">
              Prompt
            </label>
            <Textarea
              id="prompt"
              className="h-48"
              placeholder="Optional: audience, angle, structure, sources to consider, or a full one-shot writing brief"
              value={aiArticlePrompt}
              onChange={(e) => setAiArticlePrompt(e.target.value)}
            />
          </div>

          {formError && <p className="text-sm text-destructive">{formError}</p>}

          <DrawerFooter>
            <div className="w-full flex flex-row gap-4">
            <DrawerClose asChild>
              <Button className="w-1/2" variant="outline" type="button">
                Cancel
              </Button>
            </DrawerClose>
            <Button className="w-1/2" type="submit" disabled={isGenerating}>
              {isGenerating ? "Generating..." : "Generate"}
            </Button>
            </div>
          </DrawerFooter>
        </form>
      </DrawerContent>
    </Drawer>
  );
} 
