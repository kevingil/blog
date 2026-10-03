import { HeroSection } from "@/components/home/hero";
import { useQuery } from '@tanstack/react-query';
import { getArticles } from '@/services/blog';
import { type ArticleListItem, getDisplayTitle, getDisplayContent, getDisplayImage, externalArticleUrl } from '@/services/types';
import { BlurhashImage } from '@/components/media/BlurhashImage';
import { ArticleHref } from '@/components/blog/ArticleHref';
import { ExternalDomain } from '@/components/blog/ExternalDomain';
import { Link } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { format } from 'date-fns';
import { cn } from "@/lib/utils";
import { useEffect, useMemo, useRef, useState } from "react";
import GithubIcon from "@/components/icons/github-icon";
import LinkedInIcon from "@/components/icons/linkedin-icon";

export const Route = createFileRoute('/_publicLayout/')({
  component: HomePage,
});

function HomePage() {
  return (
    <div className="relative">
      <div className="relative z-10">
        <HeroSection />
        <ArticlesSection />
        <ConnectSection />
      </div>
    </div>
  );
}

/* ─── Section header helper ─── */
function SectionHeader({ label, seeAllHref, seeAllLabel = "See all" }: { label: string; seeAllHref?: string; seeAllLabel?: string }) {
  return (
    <div className="flex items-center gap-4 mb-6 px-2">
      <h2 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground whitespace-nowrap">{label}</h2>
      <div className="flex-1 h-px bg-gradient-to-r from-border to-transparent" />
      {seeAllHref && (
        <Link to={seeAllHref} className="group flex items-center gap-1.5 text-xs font-medium text-primary hover:text-primary/80 transition-colors whitespace-nowrap">
          {seeAllLabel}
          <svg className="w-3 h-3 transition-transform group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 8l4 4m0 0l-4 4m4-4H3" />
          </svg>
        </Link>
      )}
    </div>
  );
}

/* ─── Reveal a tile once it scrolls into view ─── */
function useReveal<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [shown, setShown] = useState(false);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      setShown(true);
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setShown(true);
          observer.disconnect();
        }
      },
      { threshold: 0.18, rootMargin: "0px 0px -40px 0px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return { ref, shown };
}

type TileKind = "feature" | "split" | "stack" | "overlay" | "compact" | "portrait";

const tileSpan: Record<TileKind, 2 | 3 | 4 | 6> = {
  feature: 6,
  split: 4,
  stack: 3,
  overlay: 3,
  compact: 2,
  portrait: 2,
};

/** Pack a 6-column row with a random mix of tile shapes. */
function packTiles(count: number): TileKind[] {
  const kinds: TileKind[] = [];
  let remaining = 6;
  const weights: [TileKind, number][] = [
    ["feature", 2],
    ["split", 4],
    ["stack", 3],
    ["overlay", 3],
    ["compact", 2],
    ["portrait", 2],
  ];
  for (let i = 0; i < count; i++) {
    const choices: TileKind[] = [];
    for (const [kind, weight] of weights) {
      const span = tileSpan[kind];
      if (span > remaining) continue;
      const left = remaining - span;
      if (left !== 0 && left !== 2 && left !== 3 && left !== 4) continue;
      for (let n = 0; n < weight; n++) choices.push(kind);
    }
    const kind = choices[Math.floor(Math.random() * choices.length)] ?? "compact";
    kinds.push(kind);
    remaining -= tileSpan[kind];
    if (remaining === 0) remaining = 6;
  }
  return kinds;
}

/* ─── Article helpers ─── */
function articleMeta(article: ArticleListItem) {
  const title = getDisplayTitle(article.article);
  const content = getDisplayContent(article.article);
  const image = getDisplayImage(article.article);
  const date = article.article.published_at ? new Date(article.article.published_at) : null;
  const dateStr = date && !isNaN(date.getTime()) ? format(date, 'MMM d, yyyy') : '';
  const plain = content?.replace(/<[^>]*>/g, '').replace(/\*\*/g, '').replace(/#*/g, '').replace(/\n/g, ' ').substring(0, 150) || '';
  return {
    title,
    image,
    dateStr,
    plain,
    slug: article.article.slug as string,
    author: article.author?.name,
    externalUrl: externalArticleUrl(article.article),
  };
}

const glassCard = "bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border hover:border-primary/30 hover:shadow-[0_0_20px_-5px_rgba(249,115,22,0.18)] transition-all duration-500";

/* ════════════════════════════════════════
   ARTICLES SECTION
   ════════════════════════════════════════ */
function ArticlesSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['home-articles'],
    queryFn: () => getArticles(1, null, 'published', 24),
  });

  const articles = data?.articles ?? [];
  const spanKey = articles.map((article) => article.article.id).join("|");
  const kinds = useMemo(() => packTiles(articles.length), [spanKey, articles.length]);

  return (
    <section className="mt-28 px-2 sm:px-0">
      <SectionHeader label="Articles" seeAllHref="/blog" />

      {isLoading ? (
        <ArticlesSkeleton />
      ) : articles.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground text-sm">No articles yet.</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-3">
          {articles.map((article, index) => (
            <ArticleTile
              key={article.article.id}
              article={article}
              index={index}
              kind={kinds[index] ?? "compact"}
            />
          ))}
        </div>
      )}
    </section>
  );
}

const tileSpanClass: Record<TileKind, string> = {
  feature: "md:col-span-2 lg:col-span-6",
  split: "md:col-span-2 lg:col-span-4",
  stack: "md:col-span-1 lg:col-span-3",
  overlay: "md:col-span-1 lg:col-span-3",
  compact: "md:col-span-1 lg:col-span-2",
  portrait: "md:col-span-1 lg:col-span-2",
};

function TileCover({
  image,
  className,
}: {
  image: ReturnType<typeof articleMeta>["image"];
  className?: string;
}) {
  if (image?.url) {
    return (
      <BlurhashImage
        src={image.url}
        blurhash={image.blurhash}
        className={cn("h-full w-full", className)}
        imgClassName="h-full w-full object-cover object-center transition-transform duration-500 group-hover:scale-105"
      />
    );
  }
  return (
    <div className={cn("flex h-full w-full items-center justify-center bg-muted/40", className)}>
      <svg className="h-5 w-5 text-muted-foreground/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
      </svg>
    </div>
  );
}

function TileMeta({
  author,
  externalUrl,
  dateStr,
  onDark = false,
}: {
  author?: string;
  externalUrl?: string | null;
  dateStr: string;
  onDark?: boolean;
}) {
  return (
    <div className={cn(
      "mt-auto flex flex-wrap items-center gap-x-2 gap-y-1 pt-3 text-[11px]",
      onDark ? "text-white/75" : "text-muted-foreground",
    )}>
      {author && <span className="truncate">{author}</span>}
      {externalUrl && (
        <>
          {author && <span aria-hidden>·</span>}
          <ExternalDomain url={externalUrl} className="max-w-[10rem]" iconClassName="size-2.5" />
        </>
      )}
      {dateStr && (
        <>
          {(author || externalUrl) && <span aria-hidden>·</span>}
          <span className="shrink-0">{dateStr}</span>
        </>
      )}
    </div>
  );
}

function ArticleTile({ article, index, kind }: { article: ArticleListItem; index: number; kind: TileKind }) {
  const meta = articleMeta(article);
  const { ref, shown } = useReveal<HTMLDivElement>();
  const { title, image, dateStr, plain, slug, author, externalUrl } = meta;

  return (
    <div
      ref={ref}
      className={cn(
        tileSpanClass[kind],
        "min-w-0 motion-reduce:translate-y-0 motion-reduce:opacity-100",
        "transition-[opacity,transform] duration-700 ease-out",
        shown ? "opacity-100 translate-y-0" : "opacity-0 translate-y-6",
      )}
      style={{ transitionDelay: shown ? `${(index % 4) * 70}ms` : "0ms" }}
    >
      <ArticleHref
        slug={slug}
        externalUrl={externalUrl}
        className={cn(glassCard, "group flex h-full min-h-0 overflow-hidden", tileFrameClass(kind))}
      >
        <TileBody kind={kind} title={title} image={image} plain={plain} author={author} externalUrl={externalUrl} dateStr={dateStr} />
      </ArticleHref>
    </div>
  );
}

function tileFrameClass(kind: TileKind): string {
  if (kind === "overlay") return "relative min-h-[280px] flex-col";
  if (kind === "compact") return "flex-row";
  if (kind === "feature" || kind === "split") return "flex-col lg:flex-row";
  return "flex-col";
}

function TileBody({
  kind,
  title,
  image,
  plain,
  author,
  externalUrl,
  dateStr,
}: {
  kind: TileKind;
  title: string;
  image: ReturnType<typeof articleMeta>["image"];
  plain: string;
  author?: string;
  externalUrl?: string | null;
  dateStr: string;
}) {
  if (kind === "overlay") {
    return (
      <>
        <div className="absolute inset-0">
          <TileCover image={image} />
        </div>
        <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-black/30 to-black/10" />
        <div className="relative flex min-h-[280px] flex-1 flex-col justify-end p-4 text-white">
          <h3 className="text-lg font-semibold tracking-tight line-clamp-2">{title}</h3>
          {plain && <p className="mt-1.5 text-sm text-white/80 line-clamp-2">{plain}</p>}
          <TileMeta author={author} externalUrl={externalUrl} dateStr={dateStr} onDark />
        </div>
      </>
    );
  }

  if (kind === "compact") {
    return (
      <>
        <div className="relative w-24 shrink-0 self-stretch overflow-hidden bg-muted/40 sm:w-28">
          <TileCover image={image} />
        </div>
        <div className="flex min-w-0 flex-1 flex-col p-3">
          <h3 className="text-sm font-semibold tracking-tight text-foreground transition-colors line-clamp-2 group-hover:text-primary">{title}</h3>
          {plain && <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{plain}</p>}
          <TileMeta author={author} externalUrl={externalUrl} dateStr={dateStr} />
        </div>
      </>
    );
  }

  const wide = kind === "feature" || kind === "split";
  return (
    <>
      <div
        className={cn(
          "relative shrink-0 overflow-hidden bg-muted/40",
          kind === "feature" && "aspect-[16/8] lg:aspect-auto lg:w-1/2 lg:self-stretch",
          kind === "split" && "aspect-[16/10] lg:aspect-auto lg:w-[44%] lg:self-stretch",
          kind === "portrait" && "aspect-[3/4]",
          kind === "stack" && "aspect-[16/9]",
        )}
      >
        <TileCover image={image} />
      </div>
      <div className={cn("flex min-w-0 flex-1 flex-col", kind === "feature" ? "justify-center p-6" : wide ? "p-5" : "p-4")}>
        <h3
          className={cn(
            "font-semibold tracking-tight text-foreground transition-colors line-clamp-2 group-hover:text-primary",
            kind === "feature" ? "text-2xl" : kind === "split" ? "text-lg" : kind === "stack" ? "text-base" : "text-sm",
          )}
        >
          {title}
        </h3>
        {plain && (
          <p className={cn(
            "mt-1.5 text-muted-foreground",
            (kind === "feature" || kind === "split") && "text-sm line-clamp-3",
            kind === "stack" && "text-xs line-clamp-2",
            kind === "portrait" && "text-xs line-clamp-3",
          )}>
            {plain}
          </p>
        )}
        <TileMeta author={author} externalUrl={externalUrl} dateStr={dateStr} />
      </div>
    </>
  );
}

/* ════════════════════════════════════════
   CONNECT SECTION — social link chips
   ════════════════════════════════════════ */
function ConnectSection() {
  return (
    <section className="mt-16 mb-8 px-2 sm:px-0">
      <SectionHeader label="Connect" />
      <div className="flex gap-3 flex-wrap">
        <a
          href="https://github.com/kevingil"
          target="_blank"
          className={cn(
            "group inline-flex items-center gap-2.5 px-5 py-3 rounded-xl",
            "bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border",
            "hover:border-primary/30 hover:shadow-[0_0_15px_-5px_rgba(249,115,22,0.12)] hover:scale-[1.02]",
            "transition-all duration-300"
          )}
        >
          <GithubIcon className="w-5 h-5 fill-foreground/60 group-hover:fill-primary transition-colors" />
          <span className="text-sm font-medium text-foreground/70 group-hover:text-primary transition-colors">Github</span>
        </a>
        <a
          href="https://linkedin.com/in/kevingil"
          target="_blank"
          className={cn(
            "group inline-flex items-center gap-2.5 px-5 py-3 rounded-xl",
            "bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border",
            "hover:border-primary/30 hover:shadow-[0_0_15px_-5px_rgba(249,115,22,0.12)] hover:scale-[1.02]",
            "transition-all duration-300"
          )}
        >
          <LinkedInIcon className="w-5 h-5 fill-foreground/60 group-hover:fill-primary transition-colors" />
          <span className="text-sm font-medium text-foreground/70 group-hover:text-primary transition-colors">LinkedIn</span>
        </a>
      </div>
    </section>
  );
}

/* ─── Skeleton ─── */
function ArticlesSkeleton() {
  const kinds: TileKind[] = ["feature", "split", "compact", "overlay", "portrait", "stack", "compact"];
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-3">
      {kinds.map((kind, index) => (
        <div
          key={index}
          className={cn(
            "animate-pulse overflow-hidden border border-border bg-card",
            tileSpanClass[kind],
            kind === "feature" ? "h-64" : kind === "compact" ? "h-28" : "h-64",
          )}
        >
          <div className="h-2/3 bg-muted/50" />
          <div className="space-y-2 p-4">
            <div className="h-3 w-3/4 rounded bg-muted/60" />
            <div className="h-2.5 w-full rounded bg-muted/40" />
          </div>
        </div>
      ))}
    </div>
  );
}
