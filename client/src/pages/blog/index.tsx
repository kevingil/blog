import React, { useEffect, useState } from 'react';
import { Container, Title, Loader, Pagination, Group } from '@mantine/core';
import ArticlesList from '@/components/blog/ArticleList';
import BlogService from '@/services/blog/fetchArticles';
import { ArticleData, ITEMS_PER_PAGE } from '@/services/blog/types';

const ArticlesPage: React.FC = () => {
  const [articles, setArticles] = useState<ArticleData[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalItems, setTotalItems] = useState(0);

  useEffect(() => {
    const fetchArticles = async () => {
      setLoading(true);
      try {
        const { articles, total } = await BlogService.fetchArticles({
          page: currentPage
        });
        setArticles(articles);
        setTotalItems(total);
      } catch (error) {
        console.error('Error fetching articles:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchArticles();
  }, [currentPage]);

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  // Calculate total pages using ITEMS_PER_PAGE constant from the service
  const totalPages = Math.ceil(totalItems / ITEMS_PER_PAGE);

  return (
    <Container size="lg" className="page">
      <Title order={1} mb="md" ta="left">Blog</Title>
      
      {loading ? (
        <Loader />
      ) : (
        <>
          <ArticlesList articles={articles} />
          
          <Group mt="xl">
            <Pagination
              total={totalPages}
              value={currentPage}
              onChange={handlePageChange}
              withEdges
            />
          </Group>
          
          <Group mt="sm">
            <p>
              Showing {articles.length} of {totalItems} articles
            </p>
          </Group>
        </>
      )}
    </Container>
  );
};

export default ArticlesPage;
