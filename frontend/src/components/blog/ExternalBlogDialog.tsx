import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useToast } from "@/hooks/use-toast";
import { createExternalArticle } from "@/services/blog";

export function ExternalBlogDialog() {
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const submit = async () => {
    const trimmed = url.trim();
    if (!/^https?:\/\//i.test(trimmed)) {
      setError("Enter a full http or https link.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      const created = await createExternalArticle(trimmed);
      await queryClient.invalidateQueries({ queryKey: ["articles"] });
      toast({
        title: "External blog added",
        description: created.article.draft_title,
      });
      setUrl("");
      setOpen(false);
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "Could not add that link.";
      setError(message);
    } finally {
      setPending(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setError(null);
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline">
          <ExternalLink className="mr-2 h-4 w-4" />
          External blog
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add an external blog</DialogTitle>
          <DialogDescription>
            Paste a post published elsewhere by this author. We save the link, a short preview,
            and a copy of the cover image.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-2">
          <Label htmlFor="external-blog-url">Article link</Label>
          <Input
            id="external-blog-url"
            type="url"
            placeholder="https://example.com/blog/post"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                void submit();
              }
            }}
            autoFocus
          />
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
        <DialogFooter>
          <Button onClick={() => void submit()} disabled={pending || url.trim().length === 0}>
            {pending ? "Adding…" : "Add blog"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
