import { ExternalLink } from "lucide-react";
import { articleHost } from "@/services/types";
import { cn } from "@/lib/utils";

export function ExternalDomain({
  url,
  className,
  iconClassName,
}: {
  url: string;
  className?: string;
  iconClassName?: string;
}) {
  return (
    <span className={cn("inline-flex min-w-0 items-center gap-1 text-primary", className)}>
      <span className="truncate">{articleHost(url)}</span>
      <ExternalLink className={cn("size-3 shrink-0", iconClassName)} aria-hidden />
    </span>
  );
}
