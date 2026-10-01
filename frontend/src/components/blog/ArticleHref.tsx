import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

type ArticleHrefProps = {
  slug: string;
  externalUrl?: string | null;
  className?: string;
  children: ReactNode;
};

export function ArticleHref({ slug, externalUrl, className, children }: ArticleHrefProps) {
  if (externalUrl) {
    return (
      <a href={externalUrl} target="_blank" rel="noopener noreferrer" className={className}>
        {children}
      </a>
    );
  }

  return (
    <Link
      to="/blog/$blogSlug"
      params={{ blogSlug: slug }}
      search={{ page: undefined, tag: undefined, search: undefined }}
      className={className}
    >
      {children}
    </Link>
  );
}
