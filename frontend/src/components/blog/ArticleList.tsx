'use client'

import { useEffect, useState, useCallback, useRef } from 'react';
import { useSearchParams } from 'next/navigation';
import { format } from 'date-fns';
import { Card, CardContent } from "@/components/ui/card";
import { Image as ImageIcon } from "lucide-react";
import Link from "next/link";
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
import { getArticles, searchArticles, getPopularTags } from './search';
import { ArticleListItem, ITEMS_PER_PAGE } from './index';


// Debounce delay in ms
const SEARCH_DELAY = 500; 

type ArticleListProps = {
  pagination: boolean;
}


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
  const searchParams = useSearchParams();
  const debounceTimeout = useRef<NodeJS.Timeout | null>(null);
  
  const [page, setPage] = useState(Number(searchParams.get('page')) || 1);
  const [searchTag, setSearchTag] = useState<string | null>(searchParams.get('tag'));
  const [articles, setArticles] = useState<ArticleListItem[]>([]);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState<string>(searchParams.get('search') || '');
  const [recentTags, setRecentTags] = useState<string[]>(['All']);

  // Update URL without triggering navigation, for tags, pages, and search
  const updateURLQuietly = useCallback((newParams: { page?: number; search?: string; tag?: string | null }) => {
    const params = new URLSearchParams(searchParams);

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

  }, [searchParams]);

  // On every action, query params are updated first
  // then we fetch articles based on current search params
  const fetchArticles = useCallback(async (searchValue: string, pageNum: number, tag: string | null = searchTag) => {
    setLoading(true);
    try {
      let result;
      if (searchValue) {
        result = await searchArticles(searchValue, pageNum, tag);
      } else {
        result = await getArticles(pageNum, tag);
      }
      setArticles(result.articles);
      setTotalPages(result.totalPages);
    } catch (error) {
      console.error('Error fetching articles:', error);
    } finally {
      setLoading(false);
    }
  }, []);


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
      fetchArticles(value, 1, searchTag);
    }, SEARCH_DELAY),
    [updateURLQuietly, fetchArticles]
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
    fetchArticles(searchTerm, 1, newTag);
  };

  // Handle pagination
  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    updateURLQuietly({ page: newPage });
    fetchArticles(searchTerm, newPage, searchTag);
  };

  // Initial data fetch
  useEffect(() => {
    getPopularTags().then((tags) => {
      const allTags = ['All', ...tags.tags];
      setRecentTags(allTags);
    });
    fetchArticles(searchTerm, 1, searchTag);
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
            {recentTags.map((tag) => (
              <Badge
                key={tag}
                variant={searchTag === tag || (searchTag === null && tag === 'All') ? "default" : "secondary"}
                className="cursor-pointer hover:bg-primary/90"
                onClick={() => handleTagClick(tag)}
              >
                {tag}
              </Badge>
            ))}
          </div>
        </div>
      )}


    <div ref={containerRef} className={`${animate ? 'animate-card-home' : 'hide-card-home'}`}>

      {!pagination && (
        <div className="flex justify-between p-4 items-center">
          <h2 className="font-semibold text-muted-foreground">
            Recent Articles
          </h2>
          <Link href="/blog"
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
          {articles.map((article) => (
            <Card key={article.id}>
              <CardContent className="p-0">
                <Link href={`/blog/${article.slug}`}
                  className='w-full h-full flex flex-row justify-between'>
                  <div className='p-4 w-full'>
                    <h2 className="text-xl font-semibold mb-2">{article.title}</h2>
                    <div className="flex items-center mb-4">
                      <span className="text-sm text-muted-foreground">{article.author}</span>
                    </div>
                    <div className="flex items-center mb-4 gap-2">
                      <p className="text-sm text-muted-foreground mb-4">
                        {markdownToPlainText(article.content?.substring(0, 160) || '' )}
                      </p>
                      <div className='items-start flex sm:hidden w-full min-w-[40%]'>
                        {article.image !== null && article.image !== '' ? (
                          <img src={article.image} alt={article.title ? article.title : ''}
                            className="rounded-lg object-cover aspect-square h-full w-full bg-gray-300/10 dark:bg-gray-100/10" />
                        ) : (
                          <ImageIcon className='h-2/3 w-full text-zinc-200 dark:text-zinc-600' />
                        )}
                      </div>
                    </div>
                    <p className="text-sm text-muted-foreground mb-4">
                      {article.publishedAt ? format(new Date(article.publishedAt), 'MMMM d, yyyy') : 'Unknown'}
                    </p>
                    <div className="flex flex-wrap gap-2 mb-4">
                      {article.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-primary">{tag}</Badge>
                      ))}
                    </div>
                  </div>
                  <div className='max-w-[30%] p-4 items-center hidden sm:flex aspect-square'>
                    {article.image !== null && article.image !== '' ? (
                      <img src={article.image} alt={article.title ? article.title : ''}
                        className="rounded-lg object-cover h-full w-full bg-gray-300/10 dark:bg-gray-100/10" />
                    ) : (
                      <ImageIcon className='h-2/3 w-full text-zinc-200 dark:text-zinc-600' />
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
