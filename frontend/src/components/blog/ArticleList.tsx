import { useEffect, useState, useCallback, useRef } from 'react';
import { useRouter, Link } from '@tanstack/react-router';
import { format } from 'date-fns';
import { Card, CardContent } from "@/components/ui/card";
import { Image as ImageIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { Input } from "@/components/ui/input";
import { getArticles, searchArticles, getPopularTags } from '@/services/blog';
import { ArticleListItem, ITEMS_PER_PAGE } from '@/services/types';
import { GetArticlesResponse } from '@/routes/dashboard/blog';
import { useQuery } from '@tanstack/react-query';


// Debounce delay in ms
const SEARCH_DELAY = 500; 

type ArticleListProps = {
  pagination: boolean;
}

type SearchParams = {
  page?: string;
  tag?: string;
  search?: string;
};

function ArticleCardSkeleton() {
  return (
    <Card>
      <CardContent className="p-4">
        <Skeleton className="h-6 w-3/4 mb-2" />
        <div className="flex items-center mb-4">
          <Skeleton className="h-8 w-8 rounded-full mr-2" />
          <Skeleton className="h-4 w-1/4" />
        </div>
        <Skeleton className="h-4 w-full mb-2" />
        <Skeleton className="h-4 w-5/6 mb-4" />
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-6 w-16" />
          <Skeleton className="h-6 w-20" />
        </div>
      </CardContent>
    </Card>
  );
}

export function ArticlesSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 w-full">
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <ArticleCardSkeleton key={i} />
      ))}
    </div>
  );
}

export default function ArticlesList({ pagination }: ArticleListProps) {
  const router = useRouter();
  const search = new URLSearchParams(router.state.location.search);
  const debounceTimeout = useRef<NodeJS.Timeout | null>(null);
  
  const [page, setPage] = useState(Number(search.get('page')) || 1);
  const [searchTag, setSearchTag] = useState<string | null>(search.get('tag'));
  const [searchTerm, setSearchTerm] = useState<string>(search.get('search') || '');
  const [recentTags, setRecentTags] = useState<string[]>(['All']);

  // Update URL without triggering navigation, for tags, pages, and search
  const updateURLQuietly = useCallback((newParams: { page?: number; search?: string; tag?: string | null }) => {
    const params = new URLSearchParams(search);

    if (newParams.page) {
      params.set('page', newParams.page.toString());
    }
    
    if (newParams.search !== undefined) {
      if (newParams.search) {
        params.set('search', newParams.search);
        params.delete('tag');
      } else {
        params.delete('search');
      }
    }
    
    if (newParams.tag !== undefined) {
      if (newParams.tag) {
        params.set('tag', newParams.tag);
      } else {
        params.delete('tag');
      }
    }

    window.history.replaceState({}, '', `?${params.toString()}`);

  }, [search]);

  // React Query – fetch articles based on current params
  const {
    data: articlesData,
    isLoading,
    isFetching,
  } = useQuery<GetArticlesResponse>({
    queryKey: ['public-articles', page, searchTerm, searchTag],
    queryFn: () => {
      if (searchTerm) {
        return searchArticles(searchTerm, page, searchTag);
      }
      return getArticles(page, searchTag, 'published'); // Only show published articles in public view
    },
  });

  const articles: ArticleListItem[] = (articlesData as GetArticlesResponse | undefined)?.articles ?? [];
  const totalPages: number = (articlesData as GetArticlesResponse | undefined)?.total_pages ?? 0;
  const loading = isLoading || (isFetching && !articlesData);

  // Debounce implementation function
  // thanks to: https://blog.alexefimenko.com/posts/debounce-react
  const debounce = (func: Function, delay: number) => {
    return function (...args: any[]) {
      if (debounceTimeout.current) {
        clearTimeout(debounceTimeout.current);
      }
      
      debounceTimeout.current = setTimeout(() => {
        func(...args);
        debounceTimeout.current = null;
      }, delay);
    };
  };

  // Debounce (delay) search
  const debouncedSearch = useCallback(
    debounce((value: string) => {
      setPage(1);
      setSearchTag(null);
      updateURLQuietly({ search: value, page: 1 });
      // React Query will automatically refetch when searchTerm changes
    }, SEARCH_DELAY),
    [updateURLQuietly]
  );

  // Handle search input change
  const handleSearch = (value: string) => {
    setSearchTerm(value);
    debouncedSearch(value);
  };

  // Handle tag selection
  const handleTagClick = (tag: string) => {
    const newTag = searchTag === tag ? null : tag;
    setSearchTag(newTag);
    setPage(1);
    updateURLQuietly({ tag: newTag, page: 1 });
    // React Query refetch through dependency change
  };

  // Handle pagination
  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    updateURLQuietly({ page: newPage });
  };

  // Initial data fetch
  useEffect(() => {
    getPopularTags().then((tags) => {
      const allTags = ['All', ...tags.tags];
      setRecentTags(allTags);
    });
    // React Query will handle the initial fetch automatically
  }, []); 

  // State to control the animation
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [animate, setAnimate] = useState(false);

  // Intersection Observer
  useEffect(() => {
    setAnimate(true);
  }, []);

  const markdownToPlainText = (markdown: string) => {
    return markdown
      .replace(/\*\*/g, '')     
      .replace(/#*/g, '')  
      .replace(/\n/g, ' '); 
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:py-8 w-full"
     style={{
      perspective: '20rem',
     }}>
      {pagination && (
        <div className='w-full'>
          <div className="relative">
            <Input
              type="search"
              placeholder="Search articles..."
              value={searchTerm}
              onChange={(e) => handleSearch(e.target.value)}
              className="w-full p-4 py-6 rounded-full"
            />
            {debounceTimeout.current && (
              <div className="absolute right-4 top-1/2 -translate-y-1/2">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary"></div>
              </div>
            )}
          </div>
          <div className='flex flex-wrap gap-2 my-4'>
            {recentTags && recentTags.length > 0 && recentTags.map((tag) => (
              <Badge
                key={tag}
                variant={searchTag === tag || (searchTag === null && tag === 'All') ? "default" : "secondary"}
                className="cursor-pointer hover:bg-primary/90"
                onClick={() => handleTagClick(tag)}
              >
                {tag.toUpperCase()}
              </Badge>
            ))}
          </div>
        </div>
      )}


    <div ref={containerRef}>

      {!pagination && (
        <div className="flex justify-between p-4 items-center">
          <h2 className="font-semibold text-muted-foreground">
            Recent Articles
          </h2>
          <Link to="/blog" search={{ page: undefined, tag: undefined, search: undefined }} 
            className="flex items-center font-medium text-primary transition-colors duration-200 
            border border-gray-300 dark:border-gray-800 bg-card hover:border-primary dark:hover:border-primary rounded-lg px-4 py-2 shadow-sm">
            <p className="text-md text-muted-foreground">See all</p>
          </Link>
        </div>
      )}

      {loading ? (
        <ArticlesSkeleton />
      ) : articles.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {searchTerm && !debounceTimeout.current ? 
            "No articles found matching your search criteria." :
            "Loading results..."
          }
        </div>
      ) : (
        <div className={`grid grid-cols-1 gap-4 w-full`}>
          {articles.map((article: ArticleListItem, index) => (
            <Card
              key={article.article.id}
              animationDelay={index * 100}
              className="group relative overflow-hidden hover:shadow-2xl transition-all duration-500 hover:border-primary/50"
            >
              <CardContent className="p-0">
                <Link
                  to="/blog/$blogSlug"
                  params={{ blogSlug: article.article.slug as string }}
                  search={{ page: undefined, tag: undefined, search: undefined }}
                  className="flex items-stretch gap-4"
                >
                  {/* Text */}
                  <div className="flex-1 p-4 sm:p-5">
                    <h2 className="text-lg sm:text-xl font-semibold mb-1 line-clamp-2 group-hover:text-primary transition-colors">
                      {article.article?.title}
                    </h2>
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs sm:text-sm text-muted-foreground">{article.author?.name}</span>
                      <span className="text-xs text-muted-foreground">
                        {(() => {
                          const date = article.article.published_at ? new Date(article.article.published_at) : null;
                          return date && !isNaN(date.getTime()) ? format(date, 'MMM d, yyyy') : 'Unknown';
                        })()}
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {markdownToPlainText(article.article.content?.substring(0, 200) || '')}
                    </p>
                    {article.tags && article.tags.length > 0 && (
                      <div className="flex flex-wrap gap-2 mt-3">
                        {article.tags.slice(0, 3).map((tag) => (
                          tag.name ? (
                            <Badge key={tag.tag_id} variant="secondary" className="text-primary">
                              {tag.name.toUpperCase()}
                            </Badge>
                          ) : null
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Image */}
                  <div className="relative w-36 sm:w-48 md:w-56 flex-shrink-0 overflow-hidden rounded-md my-4 mr-4">
                    {article.article.image_url ? (
                      <>
                        <img
                          src={article.article.image_url}
                          alt={article.article.title ? article.article.title : ''}
                          className="w-full h-full object-cover aspect-video transition-transform duration-300 group-hover:scale-110"
                        />
                        <div className="absolute inset-0 bg-gradient-to-t from-black/20 via-transparent to-transparent" />
                      </>
                    ) : (
                      <div className="w-full h-full aspect-video bg-muted flex items-center justify-center">
                        <ImageIcon className="w-10 h-10 text-muted-foreground" />
                      </div>
                    )}
                  </div>
                </Link>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
      </div>

      {pagination && totalPages > 1 && (
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                onClick={() => page > 1 && handlePageChange(page - 1)}
                className={page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
              />
            </PaginationItem>
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
              <PaginationItem key={pageNumber}>
                <PaginationLink
                  onClick={() => handlePageChange(pageNumber)}
                  isActive={pageNumber === page}
                  className="cursor-pointer"
                >
                  {pageNumber}
                </PaginationLink>
              </PaginationItem>
            ))}
            <PaginationItem>
              <PaginationNext
                onClick={() => page < totalPages && handlePageChange(page + 1)}
                className={page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  );
}
