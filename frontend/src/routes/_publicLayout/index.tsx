import { HeroSection } from "@/components/home/hero";
import { useQuery } from '@tanstack/react-query';
import { listProjects, type Project } from '@/services/projects';
import { getArticles } from '@/services/blog';
import { type ArticleListItem, getDisplayTitle, getDisplayContent, getDisplayImageUrl } from '@/services/types';
import { Link } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { format } from 'date-fns';
import { cn } from "@/lib/utils";
import { useState, useEffect } from "react";
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
        <ProjectsSection />
        <ConnectSection />
      </div>
    </div>
  );
}

/* ─── Section header helper ─── */
function SectionHeader({ label, seeAllHref, seeAllLabel = "See all" }: { label: string; seeAllHref?: string; seeAllLabel?: string }) {
  return (
    <div className="flex items-center gap-4 mb-6 px-2">
      <h2 className="text-xs font-semibold uppercase tracking-widest text-white/40 whitespace-nowrap">{label}</h2>
      <div className="flex-1 h-px bg-gradient-to-r from-white/10 to-transparent" />
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

/* ─── Staggered entrance hook ─── */
function useEntrance(index: number) {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const t = setTimeout(() => setVisible(true), 100 + index * 80);
    return () => clearTimeout(t);
  }, [index]);
  return visible;
}

/* ─── Article helpers ─── */
function articleMeta(article: ArticleListItem) {
  const title = getDisplayTitle(article.article);
  const content = getDisplayContent(article.article);
  const imageUrl = getDisplayImageUrl(article.article);
  const date = article.article.published_at ? new Date(article.article.published_at) : null;
  const dateStr = date && !isNaN(date.getTime()) ? format(date, 'MMM d, yyyy') : '';
  const plain = content?.replace(/<[^>]*>/g, '').replace(/\*\*/g, '').replace(/#*/g, '').replace(/\n/g, ' ').substring(0, 150) || '';
  return { title, imageUrl, dateStr, plain, slug: article.article.slug as string, author: article.author?.name };
}

const glassCard = "bg-black/30 dark:bg-black/40 backdrop-blur-md border border-white/[0.08] rounded-2xl hover:border-primary/30 hover:shadow-[0_0_20px_-5px_rgba(249,115,22,0.1)] transition-all duration-500";

/* ════════════════════════════════════════
   ARTICLES SECTION
   ════════════════════════════════════════ */
function ArticlesSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['home-articles'],
    queryFn: () => getArticles(1, null, 'published', 12),
  });

  const articles = data?.articles ?? [];
  const mainArticle = articles[0];
  const compactArticles = articles.slice(1, 3);
  const listArticles = articles.slice(3);

  return (
    <section className="mt-28 px-2 sm:px-0">
      <SectionHeader label="Articles" seeAllHref="/blog" />

      {isLoading ? (
        <ArticlesSkeleton />
      ) : articles.length === 0 ? (
        <div className="text-center py-16 text-white/30 text-sm">No articles yet.</div>
      ) : (
        <>
          {/* Zone 1: Bento — 1 main + 2 compact */}
          {(mainArticle || compactArticles.length > 0) && (
            <div className="grid grid-cols-1 lg:grid-cols-3 lg:grid-rows-2 gap-2 mb-3">
              {mainArticle && (
                <MainArticleCard key={mainArticle.article.id} article={mainArticle} index={0} />
              )}
              {compactArticles.map((article, i) => (
                <CompactArticleCard key={article.article.id} article={article} index={i + 1} />
              ))}
            </div>
          )}

          {/* Zone 2: Full article list */}
          {listArticles.length > 0 && (
            <div className="rounded-xl bg-black/20 backdrop-blur-sm border border-white/[0.05] overflow-hidden">
              {listArticles.map((article, i) => {
                const { title, imageUrl, dateStr, slug, plain } = articleMeta(article);
                return (
                  <Link
                    key={article.article.id}
                    to="/blog/$blogSlug"
                    params={{ blogSlug: slug }}
                    className={cn(
                      "flex items-center gap-3 px-3 py-2.5 group hover:bg-white/[0.04] transition-colors",
                      i < listArticles.length - 1 && "border-b border-white/[0.04]"
                    )}
                  >
                    <div className="w-14 h-10 shrink-0 rounded-lg overflow-hidden bg-white/[0.03]">
                      {imageUrl ? (
                        <img src={imageUrl} alt="" className="w-full h-full object-cover object-center transition-transform duration-200 group-hover:scale-105" />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center">
                          <svg className="w-3.5 h-3.5 text-white/15" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
                          </svg>
                        </div>
                      )}
                    </div>
                    <div className="flex-1 min-w-0 flex flex-col gap-0.5">
                      <span className="text-sm text-white/55 group-hover:text-primary transition-colors truncate">{title}</span>
                      {plain && (
                        <span className="text-[11px] text-white/30 line-clamp-1">{plain}</span>
                      )}
                    </div>
                    <span className="text-[11px] text-white/20 shrink-0">{dateStr}</span>
                  </Link>
                );
              })}
            </div>
          )}
        </>
      )}
    </section>
  );
}

function MainArticleCard({ article, index }: { article: ArticleListItem; index: number }) {
  const { title, imageUrl, dateStr, plain, slug, author } = articleMeta(article);
  const visible = useEntrance(index);

  return (
    <Link
      to="/blog/$blogSlug"
      params={{ blogSlug: slug }}
      className={cn(
        glassCard,
        "group flex flex-row overflow-hidden p-2.5 gap-3 lg:col-span-2 lg:row-span-2",
        visible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-3"
      )}
    >
      <div className="relative w-20 shrink-0 aspect-[3/2] lg:w-64 lg:self-stretch lg:min-h-0 lg:aspect-auto overflow-hidden rounded-lg">
        {imageUrl ? (
          <img src={imageUrl} alt="" className="w-full h-full object-cover object-center transition-transform duration-300 group-hover:scale-105" loading="eager" />
        ) : (
          <div className="w-full h-full bg-white/[0.03] flex items-center justify-center">
            <svg className="w-5 h-5 text-white/15" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
            </svg>
          </div>
        )}
      </div>
      <div className="flex-1 flex flex-col min-w-0 justify-center">
        <h3 className="text-xs font-semibold tracking-tight text-white group-hover:text-primary transition-colors line-clamp-2">{title}</h3>
        <p className="text-[10px] text-white/40 line-clamp-1 mt-0.5">{plain}</p>
        <div className="flex items-center gap-2 text-[10px] text-white/25 mt-1">
          {author && <span>{author}</span>}
          {dateStr && <><span>·</span><span>{dateStr}</span></>}
        </div>
      </div>
    </Link>
  );
}

function CompactArticleCard({ article, index }: { article: ArticleListItem; index: number }) {
  const { title, imageUrl, dateStr, plain, slug, author } = articleMeta(article);
  const visible = useEntrance(index);

  return (
    <Link
      to="/blog/$blogSlug"
      params={{ blogSlug: slug }}
      className={cn(
        glassCard,
        "group flex flex-row overflow-hidden p-2.5 gap-3 lg:col-start-3",
        visible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-3"
      )}
    >
      <div className="relative w-20 shrink-0 aspect-[3/2] overflow-hidden rounded-lg">
        {imageUrl ? (
          <img src={imageUrl} alt="" className="w-full h-full object-cover object-center transition-transform duration-300 group-hover:scale-105" loading="eager" />
        ) : (
          <div className="w-full h-full bg-white/[0.03] flex items-center justify-center">
            <svg className="w-3 h-3 text-white/15" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14" />
            </svg>
          </div>
        )}
      </div>
      <div className="flex-1 flex flex-col min-w-0">
        <h3 className="text-[11px] font-semibold tracking-tight text-white group-hover:text-primary transition-colors line-clamp-2">{title}</h3>
        <p className="text-[10px] text-white/35 line-clamp-1 mt-0.5">{plain}</p>
        <div className="flex items-center gap-2 text-[10px] text-white/20 mt-1">
          {author && <span>{author}</span>}
          {dateStr && <><span>·</span><span>{dateStr}</span></>}
        </div>
      </div>
    </Link>
  );
}

/* ════════════════════════════════════════
   PROJECTS SECTION — book cards
   ════════════════════════════════════════ */
function ProjectsSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['home-projects'],
    queryFn: () => listProjects(1, 8),
  });

  const projects = data?.projects ?? [];

  return (
    <section className="mt-16 px-2 sm:px-0">
      <SectionHeader label="Hackathon Projects & Experiments" seeAllHref="/projects" seeAllLabel="View all" />

      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-4 gap-1.5 w-full">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="overflow-hidden bg-white/[0.03] border border-white/[0.05] animate-pulse">
              <div className="aspect-[2/1] bg-white/[0.06]" />
              <div className="p-1 h-11 flex flex-col gap-1">
                <div className="h-3 w-4/5 bg-white/[0.06]" />
                <div className="h-2.5 w-full bg-white/[0.04]" />
              </div>
            </div>
          ))}
        </div>
      ) : projects.length === 0 ? (
        <div className="text-center py-12 text-white/30 text-sm">No hackathon projects or experiments yet.</div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-4 gap-1.5 w-full">
          {projects.map((project, i) => (
            <ProjectBookCard key={project.id} project={project} index={i} />
          ))}
        </div>
      )}
    </section>
  );
}

function ProjectBookCard({ project, index }: { project: Project; index: number }) {
  const visible = useEntrance(index);

  return (
    <Link
      to="/projects/$projectId"
      params={{ projectId: project.id }}
      className={cn(
        "group relative flex flex-col overflow-hidden",
        "bg-black/40 backdrop-blur-md border border-white/[0.08]",
        "hover:border-primary hover:shadow-[0_0_25px_rgba(249,115,22,0.5)]",
        "transition-all duration-200",
        visible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-3"
      )}
    >
      <div className="relative aspect-[2/1] w-full overflow-hidden shrink-0">
        {project.image_url ? (
          <img src={project.image_url} alt={project.title} className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" />
        ) : (
          <div className="w-full h-full bg-white/[0.04] flex items-center justify-center">
            <svg className="w-6 h-6 text-white/15" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
            </svg>
          </div>
        )}
        {project.url && (
          <div className="absolute top-1 right-1 w-4 h-4 bg-black/60 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
            <svg className="w-1.5 h-1.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </div>
        )}
      </div>
      <div className="p-1 flex flex-col min-w-0 h-11">
        <h3 className="text-xs font-semibold tracking-tight text-white truncate group-hover:text-primary transition-colors">{project.title}</h3>
        {project.description && (
          <p className="text-[10px] text-white/50 line-clamp-1 mt-0.5 flex-shrink-0">{project.description}</p>
        )}
      </div>
    </Link>
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
            "bg-black/30 dark:bg-black/40 backdrop-blur-md border border-white/[0.08]",
            "hover:border-primary/30 hover:shadow-[0_0_15px_-5px_rgba(249,115,22,0.12)] hover:scale-[1.02]",
            "transition-all duration-300"
          )}
        >
          <GithubIcon className="w-5 h-5 fill-white/60 group-hover:fill-primary transition-colors" />
          <span className="text-sm font-medium text-white/60 group-hover:text-primary transition-colors">Github</span>
        </a>
        <a
          href="https://linkedin.com/in/kevingil"
          target="_blank"
          className={cn(
            "group inline-flex items-center gap-2.5 px-5 py-3 rounded-xl",
            "bg-black/30 dark:bg-black/40 backdrop-blur-md border border-white/[0.08]",
            "hover:border-primary/30 hover:shadow-[0_0_15px_-5px_rgba(249,115,22,0.12)] hover:scale-[1.02]",
            "transition-all duration-300"
          )}
        >
          <LinkedInIcon className="w-5 h-5 fill-white/60 group-hover:fill-primary transition-colors" />
          <span className="text-sm font-medium text-white/60 group-hover:text-primary transition-colors">LinkedIn</span>
        </a>
      </div>
    </section>
  );
}

/* ─── Skeleton ─── */
function ArticlesSkeleton() {
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 lg:grid-cols-3 lg:grid-rows-2 gap-2">
        {/* Main article placeholder - 2 cols, 2 rows on desktop; compact on mobile */}
        <div className="lg:col-span-2 lg:row-span-2 rounded-xl bg-white/[0.03] border border-white/[0.05] overflow-hidden animate-pulse flex flex-row p-2.5 gap-3">
          <div className="w-20 aspect-[3/2] shrink-0 lg:w-64 lg:self-stretch lg:min-h-0 lg:aspect-auto rounded-lg bg-white/[0.06]" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3 w-3/4 bg-white/[0.06] rounded" />
            <div className="h-2.5 w-full bg-white/[0.04] rounded" />
            <div className="h-2.5 w-1/3 bg-white/[0.04] rounded" />
          </div>
        </div>
        {/* Compact 1 */}
        <div className="lg:col-start-3 rounded-xl bg-white/[0.03] border border-white/[0.05] overflow-hidden animate-pulse flex flex-row p-2.5 gap-3">
          <div className="w-20 aspect-[3/2] shrink-0 rounded-lg bg-white/[0.06]" />
          <div className="flex-1 space-y-1.5">
            <div className="h-2.5 w-3/4 bg-white/[0.06] rounded" />
            <div className="h-2 w-full bg-white/[0.04] rounded" />
          </div>
        </div>
        {/* Compact 2 */}
        <div className="lg:col-start-3 lg:row-start-2 rounded-xl bg-white/[0.03] border border-white/[0.05] overflow-hidden animate-pulse flex flex-row p-2.5 gap-3">
          <div className="w-20 aspect-[3/2] shrink-0 rounded-lg bg-white/[0.06]" />
          <div className="flex-1 space-y-1.5">
            <div className="h-2.5 w-3/4 bg-white/[0.06] rounded" />
            <div className="h-2 w-full bg-white/[0.04] rounded" />
          </div>
        </div>
      </div>
      <div className="rounded-xl bg-white/[0.02] border border-white/[0.05] overflow-hidden">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 px-3 py-2.5 border-b border-white/[0.03] animate-pulse">
            <div className="w-14 h-10 shrink-0 bg-white/[0.06] rounded-lg" />
            <div className="flex-1 flex flex-col gap-1">
              <div className="h-3.5 w-3/4 bg-white/[0.05] rounded" />
              <div className="h-3 w-full bg-white/[0.03] rounded" />
            </div>
            <div className="h-3 w-16 shrink-0 bg-white/[0.04] rounded" />
          </div>
        ))}
      </div>
    </div>
  );
}
