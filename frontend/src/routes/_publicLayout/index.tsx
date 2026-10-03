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

function articleMeta(article: ArticleListItem) {
  const title = getDisplayTitle(article.article);
  const content = getDisplayContent(article.article);
  const image = getDisplayImage(article.article);
  const date = article.article.published_at ? new Date(article.article.published_at) : null;
  const dateStr = date && !isNaN(date.getTime()) ? format(date, 'MMM d, yyyy') : '';
  const plain = content?.replace(/<[^>]*>/g, '').replace(/\*\*/g, '').replace(/#*/g, '').replace(/\n/g, ' ').substring(0, 180) || '';
  return {
    title,
    image,
    dateStr,
    plain,
    slug: article.article.slug as string,
    author: article.author?.name,
    externalUrl: externalArticleUrl(article.article),
    tags: (article.tags ?? []).map((tag) => tag.name).filter((name) => name.length > 0),
  };
}

const glassCard = "bg-card/90 dark:bg-card/80 backdrop-blur-md border border-border hover:border-primary/30 hover:shadow-[0_0_20px_-5px_rgba(249,115,22,0.18)] transition-all duration-500";

function Cover({
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
    <div className="flex h-full w-full items-center justify-center bg-muted/40">
      <svg className="h-4 w-4 text-muted-foreground/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
      </svg>
    </div>
  );
}

function ArticleMetaLine({
  author,
  externalUrl,
  dateStr,
}: {
  author?: string;
  externalUrl?: string | null;
  dateStr: string;
}) {
  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-muted-foreground">
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

function ArticlesSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['home-articles'],
    queryFn: () => getArticles(1, null, 'published', 7),
  });

  const articles = data?.articles ?? [];
  const [lead, ...rest] = articles;

  return (
    <section className="mt-28 px-2 sm:px-0">
      <SectionHeader label="Articles" seeAllHref="/blog" />

      {isLoading ? (
        <ArticlesSkeleton />
      ) : articles.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground text-sm">No articles yet.</div>
      ) : (
        <div className="flex flex-col gap-2">
          {lead && <LeadArticle article={lead} />}
          {rest.length > 0 && (
            <div className="overflow-hidden border border-border bg-card/80 backdrop-blur-sm">
              {rest.map((article, index) => (
                <ArticleRow key={article.article.id} article={article} bordered={index < rest.length - 1} />
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}

function LeadArticle({ article }: { article: ArticleListItem }) {
  const { title, image, dateStr, plain, slug, author, externalUrl } = articleMeta(article);
  return (
    <ArticleHref
      slug={slug}
      externalUrl={externalUrl}
      className={cn(glassCard, "group flex items-center gap-4 p-3")}
    >
      <div className="relative h-20 w-32 shrink-0 overflow-hidden bg-muted/40 sm:h-24 sm:w-40">
        <Cover image={image} />
      </div>
      <div className="min-w-0 flex-1">
        <h3 className="text-lg font-semibold tracking-tight text-foreground line-clamp-2 transition-colors group-hover:text-primary">
          {title}
        </h3>
        {plain && <p className="mt-1 text-sm text-muted-foreground line-clamp-2">{plain}</p>}
        <ArticleMetaLine author={author} externalUrl={externalUrl} dateStr={dateStr} />
      </div>
    </ArticleHref>
  );
}

function ArticleTags({ tags }: { tags: string[] }) {
  if (tags.length === 0) return null;
  return (
    <div className="mt-auto flex max-w-[9rem] flex-wrap justify-end gap-1 pt-1">
      {tags.slice(0, 3).map((tag) => (
        <span key={tag} className="text-[10px] uppercase tracking-wide text-muted-foreground/80">
          {tag}
        </span>
      ))}
    </div>
  );
}

function ArticleRow({ article, bordered }: { article: ArticleListItem; bordered: boolean }) {
  const { title, image, dateStr, plain, slug, author, externalUrl, tags } = articleMeta(article);
  return (
    <ArticleHref
      slug={slug}
      externalUrl={externalUrl}
      className={cn(
        "group flex items-start gap-3 px-3 py-2.5 hover:bg-accent transition-colors",
        bordered && "border-b border-border",
      )}
    >
      <div className="relative mt-0.5 h-10 w-14 shrink-0 overflow-hidden bg-muted/40">
        <Cover image={image} />
      </div>
      <div className="min-w-0 flex-1">
        <h3 className="truncate text-sm text-foreground/90 transition-colors group-hover:text-primary">
          {title}
        </h3>
        {plain && <p className="truncate text-xs text-muted-foreground">{plain}</p>}
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 text-[11px] text-muted-foreground sm:hidden">
          {author && <span className="truncate">{author}</span>}
          {externalUrl && <ExternalDomain url={externalUrl} className="max-w-[10rem]" iconClassName="size-2.5" />}
          {dateStr && <span className="shrink-0">{dateStr}</span>}
        </div>
        {(author || externalUrl) && (
          <div className="mt-0.5 hidden items-center gap-x-2 text-[11px] text-muted-foreground sm:flex">
            {author && <span className="truncate">{author}</span>}
            {externalUrl && (
              <>
                {author && <span aria-hidden>·</span>}
                <ExternalDomain url={externalUrl} className="max-w-[10rem]" iconClassName="size-2.5" />
              </>
            )}
          </div>
        )}
        <div className="mt-1 flex justify-end sm:hidden">
          <ArticleTags tags={tags} />
        </div>
      </div>
      <div className="hidden min-h-full shrink-0 flex-col items-end self-stretch sm:flex">
        {dateStr && <span className="text-[11px] text-muted-foreground">{dateStr}</span>}
        <ArticleTags tags={tags} />
      </div>
    </ArticleHref>
  );
}

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

function ArticlesSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex animate-pulse items-center gap-4 border border-border bg-card p-3">
        <div className="h-24 w-40 shrink-0 bg-muted/60" />
        <div className="flex-1 space-y-2">
          <div className="h-4 w-1/2 rounded bg-muted/60" />
          <div className="h-3 w-full rounded bg-muted/40" />
          <div className="h-3 w-1/3 rounded bg-muted/40" />
        </div>
      </div>
      <div className="overflow-hidden border border-border bg-card">
        {Array.from({ length: 6 }).map((_, index) => (
          <div key={index} className="flex animate-pulse items-center gap-3 border-b border-border px-3 py-2.5 last:border-b-0">
            <div className="h-10 w-14 shrink-0 bg-muted/60" />
            <div className="flex-1 space-y-1.5">
              <div className="h-3.5 w-2/3 rounded bg-muted/60" />
              <div className="h-3 w-full rounded bg-muted/40" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
