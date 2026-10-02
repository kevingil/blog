import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { generateBlurhash, type GeneratedBlurhash } from "@/services/storage";
import { cn } from "@/lib/utils";

export function GenerateBlurhash({
  fileKey,
  uploadId,
  blurhash,
  testId,
  className,
  onGenerated,
}: {
  fileKey?: string;
  uploadId?: string;
  blurhash?: string | null;
  testId?: string;
  className?: string;
  onGenerated?: (result: GeneratedBlurhash) => void;
}) {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [hash, setHash] = useState(blurhash ?? "");
  const [pending, setPending] = useState(false);
  const canGenerate = Boolean(fileKey || uploadId);

  useEffect(() => {
    setHash(blurhash ?? "");
  }, [blurhash]);

  const handleGenerate = async () => {
    if (!canGenerate) return;
    setPending(true);
    try {
      const result = await generateBlurhash({ key: fileKey, id: uploadId });
      setHash(result.blurhash);
      onGenerated?.(result);
      await queryClient.invalidateQueries({ queryKey: ["storage", "files"] });
      toast({ title: "Blurhash generated" });
    } catch (error) {
      toast({
        title: "Could not generate blurhash",
        description: error instanceof Error ? error.message : "Try again.",
        variant: "destructive",
      });
    } finally {
      setPending(false);
    }
  };

  return (
    <div className={cn("space-y-2", className)}>
      {hash ? (
        <p className="font-mono text-xs text-muted-foreground break-all" data-testid={testId}>
          blurhash {hash}
        </p>
      ) : null}
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={!canGenerate || pending}
        onClick={handleGenerate}
      >
        {pending ? "Generating..." : hash ? "Regenerate blurhash" : "Generate blurhash"}
      </Button>
    </div>
  );
}
