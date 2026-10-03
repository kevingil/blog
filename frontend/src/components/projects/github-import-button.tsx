import { useEffect, useState } from "react";
import { Download, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { importProjectFromGithub, type GithubProjectImport } from "@/services/projects";

type GithubImportButtonProps = {
  initialUrl?: string;
  onImported: (project: GithubProjectImport) => void;
};

export function GithubImportButton({ initialUrl = "", onImported }: GithubImportButtonProps) {
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState(initialUrl);
  const [error, setError] = useState("");
  const [importing, setImporting] = useState(false);

  useEffect(() => {
    if (open) {
      setUrl(initialUrl);
      setError("");
    }
  }, [open, initialUrl]);

  const importRepository = async () => {
    const repositoryUrl = url.trim();
    if (!repositoryUrl) {
      setError("Enter a GitHub repository URL");
      return;
    }

    setImporting(true);
    setError("");
    try {
      const project = await importProjectFromGithub(repositoryUrl);
      onImported(project);
      setOpen(false);
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "Could not import that repository";
      setError(message);
    } finally {
      setImporting(false);
    }
  };

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        className="text-red-500 hover:bg-red-500/10 hover:text-red-400"
        onClick={() => setOpen(true)}
      >
        Import from Github
        <Download />
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Import from Github</DialogTitle>
            <DialogDescription>
              Paste a public repository URL. The title, description, README, tags, project URL, and first image are filled into this form.
            </DialogDescription>
          </DialogHeader>
          <form
            className="space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              void importRepository();
            }}
          >
            <Input
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="https://github.com/owner/repo"
              autoFocus
              aria-label="GitHub repository URL"
            />
            {error && <p className="text-sm text-red-500">{error}</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={importing}>
                Cancel
              </Button>
              <Button type="submit" disabled={importing}>
                {importing && <Loader2 className="animate-spin" />}
                Import
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function ProjectImagePreview({ url }: { url?: string }) {
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [url]);

  if (!url || failed || !isHttpUrl(url)) {
    return null;
  }

  return (
    <img
      src={url}
      alt="Project"
      className="mt-2 max-h-56 w-full rounded-md border object-contain"
      onError={() => setFailed(true)}
    />
  );
}

function isHttpUrl(value: string) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}
