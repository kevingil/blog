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
import { Article } from '@/services/types';

interface GenerateArticleDrawerProps {
  children: React.ReactNode;
}

export function GenerateArticleDrawer({ children }: GenerateArticleDrawerProps) {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [isGenerating, setIsGenerating] = useState(false);
  const [aiArticleTitle, setAiArticleTitle] = useState<string>('');
  const [aiArticlePrompt, setAiArticlePrompt] = useState<string>('');

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsGenerating(true);
  
    try {
      if (!user?.id) {
        throw new Error("User not found");
      }
      const newGeneratedArticle: Article = await generateArticle(aiArticlePrompt, aiArticleTitle, user.id);
      navigate({ to: `/dashboard/blog/edit/${newGeneratedArticle.slug}` });
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
            <p className="pb-4 text-gray-500 text-[0.9rem]">Both the title and prompt will be used to generate the article</p>
            <label htmlFor="title" className="block font-bold text-gray-500 text-sm mb-2">
              Title
            </label>
            <Input
              id="title"
              type="text"
              placeholder="Title will be used by the AI to generate the article"
              value={aiArticleTitle}
              onChange={(e) => setAiArticleTitle(e.target.value)}
              required
            />
          </div>

          <div>
            <label htmlFor="prompt" className="block font-bold text-gray-500 text-sm mb-2">
              Prompt
            </label>
            <Textarea
              id="prompt"
              className="h-48"
              placeholder="Addidional instructions"
              value={aiArticlePrompt}
              onChange={(e) => setAiArticlePrompt(e.target.value)}
              required
            />
          </div>

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
