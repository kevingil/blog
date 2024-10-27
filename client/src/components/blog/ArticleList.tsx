import { useEffect, useState, useCallback, useRef } from 'react';
import { format } from 'date-fns';
import { Card, Text, Group, Button, Loader, TextInput, Badge } from '@mantine/core';
import { useDispatch, useSelector } from 'react-redux';
import axios from 'axios';
import { useLocation, useNavigate } from 'react-router-dom'; // Change here
import { ArticleListItem, ITEMS_PER_PAGE } from './index';

// Debounce delay in ms
const SEARCH_DELAY = 500;

type ArticleListProps = {
  pagination: boolean;
};

function ArticleCardSkeleton() {
  return (
    <Card>
      <TextSkeleton />
      <Loader />
    </Card>
  );
}

function TextSkeleton() {
  return (
    <>
      <Text size="lg" style={{ height: '20px', width: '75%' }} />
      <Group gap="xs">
        <Loader size="sm" />
        <Text style={{ height: '15px', width: '25%' }} />
      </Group>
      <Text style={{ height: '15px', width: '100%' }} />
      <Text style={{ height: '15px', width: '80%' }} />
    </>
  );
}

export function ArticlesSkeleton() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '16px' }}>
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <ArticleCardSkeleton key={i} />
      ))}
    </div>
  );
}

export default function ArticlesList({ pagination }: ArticleListProps) {
  const location = useLocation();
  const navigate = useNavigate(); // Change here
  const debounceTimeout = useRef<NodeJS.Timeout | null>(null);
  const dispatch = useDispatch();

  const [page, setPage] = useState(new URLSearchParams(location.search).get('page') ? Number(new URLSearchParams(location.search).get('page')) : 1);
  const [searchTag, setSearchTag] = useState<string | null>(new URLSearchParams(location.search).get('tag'));
  const [articles, setArticles] = useState<ArticleListItem[]>([]);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState<string>(new URLSearchParams(location.search).get('search') || '');
  const [recentTags, setRecentTags] = useState<string[]>([]);

  const updateURLQuietly = useCallback((newParams: { page?: number; search?: string; tag?: string | null }) => {
    const params = new URLSearchParams(location.search);
    
    if (newParams.page) {
      params.set('page', newParams.page.toString());
    }
    
    if (newParams.search !== undefined) {
      if (newParams.search) {
        params.set('search', newParams.search);
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

    navigate({ search: params.toString() }); // Change here
  }, [location.search, navigate]);

  const fetchArticles = useCallback(async (searchValue: string = searchTerm, pageNum: number = page) => {
    setLoading(true);
    try {
      const result = searchValue
        ? await axios.get(`/api/articles/search`, { params: { search: searchValue, page: pageNum } })
        : await axios.get(`/api/articles`, { params: { page: pageNum } });

      setArticles(result.data.articles);
      setTotalPages(result.data.totalPages);
    } catch (error) {
      console.error('Error fetching articles:', error);
    } finally {
      setLoading(false);
    }
  }, [searchTerm, page]);

  useEffect(() => {
    setRecentTags(['React', 'Axios', 'JavaScript', 'Web Development']);
  }, []);

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

  const debouncedSearch = useCallback(
    debounce((value: string) => {
      setPage(1);
      updateURLQuietly({ search: value, page: 1 });
      fetchArticles(value, 1);
    }, SEARCH_DELAY),
    [updateURLQuietly, fetchArticles]
  );

  const handleSearch = (value: string) => {
    setSearchTerm(value);
    debouncedSearch(value);
  };

  const handleTagClick = (tag: string) => {
    const newTag = searchTag === tag ? null : tag;
    setSearchTag(newTag);
    setPage(1);
    updateURLQuietly({ tag: newTag, page: 1 });
    fetchArticles();
  };

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    updateURLQuietly({ page: newPage });
    fetchArticles(searchTerm, newPage);
  };

  useEffect(() => {
    fetchArticles();
  }, []); 

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '16px', padding: '32px 0' }}>
      {pagination && (
        <div>
          <div style={{ position: 'relative' }}>
            <TextInput
              placeholder="Search articles..."
              value={searchTerm}
              onChange={(e) => handleSearch(e.currentTarget.value)}
              style={{ width: '100%', padding: '16px', borderRadius: '50px' }}
            />
            {debounceTimeout.current && (
              <div style={{ position: 'absolute', right: '16px', top: '50%', transform: 'translateY(-50%)' }}>
                <Loader size="sm" />
              </div>
            )}
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', margin: '16px 0' }}>
            {recentTags.map((tag) => (
              <Badge
                key={tag}
                color={searchTag === tag ? 'blue' : 'gray'}
                style={{ cursor: 'pointer' }}
                onClick={() => handleTagClick(tag)}
              >
                {tag}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {!pagination && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px' }}>
          <Text size="lg">Recent Articles</Text>
          <Button component="a" href="/blog" variant="outline">See all</Button>
        </div>
      )}

      {loading ? (
        <ArticlesSkeleton />
      ) : articles.length === 0 ? (
        <Text py="lg" color="dimmed">
          {searchTerm && !debounceTimeout.current ? 
            "No articles found matching your search criteria." :
            "Loading results..."
          }
        </Text>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '16px' }}>
          {articles.map((article) => (
            <Card key={article.id}>
              <a href={`/blog/${article.slug}`} style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between', width: '100%', textDecoration: 'none' }}>
                <div style={{ padding: '16px', width: '100%' }}>
                  <Text size="lg" mb="xs">{article.title}</Text>
                  <Group gap="xs">
                    <Text size="sm" color="dimmed">{article.author}</Text>
                  </Group>
                  <Text size="sm" color="dimmed" mb="xs">{article.content?.substring(0, 160)}</Text>
                  <Text size="sm" color="dimmed" mb="xs">{format(new Date(article.createdAt), 'MMMM d, yyyy')}</Text>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px', marginBottom: '16px' }}>
                    {article.tags.map((tag) => (
                      <Badge key={tag} color="gray" variant="outline">{tag}</Badge>
                    ))}
                  </div>
                </div>
                <div style={{ width: '30%', height: '100%', position: 'relative' }}>
                  <img src={article.image ?? ''} alt={article.title ?? ''} style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: '8px' }} />
                </div>
              </a>
            </Card>
          ))}
        </div>
      )}

      {pagination && (
        <div style={{ display: 'flex', justifyContent: 'center', marginTop: '16px' }}>
          {Array.from({ length: totalPages }, (_, index) => (
            <Button
              key={index}
              variant="outline"
              onClick={() => handlePageChange(index + 1)}
              style={{ margin: '0 4px' }}
            >
              {index + 1}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}
