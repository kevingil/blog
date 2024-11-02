import React, { useEffect, useState } from 'react';
import { Container, Title, Loader, Pagination, Group } from '@mantine/core';
import ArticlesList from '@/components/blog/ArticleList';
import { fetchArticlesApi } from '@/api/articles'; 

const ArticlesPage: React.FC = () => {
  const [articles, setArticles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);
  const [totalItems, setTotalItems] = useState(0);

  const totalPages = Math.ceil(totalItems / itemsPerPage);

  useEffect(() => {
    const fetchArticles = async () => {
      setLoading(true);
      try {
        const { articles, totalItems } = await fetchArticlesApi(currentPage, itemsPerPage);
        setArticles(articles);
        setTotalItems(totalItems);
      } catch (error) {
        console.error('Error fetching articles:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchArticles();
  }, [currentPage, itemsPerPage]);

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  const handleItemsPerPageChange = (value: string) => {
    setItemsPerPage(Number(value));
    setCurrentPage(1); // Reset to first page when changing items per page
  };

  return (
    <Container size="lg" className="page">
      <Title order={1} mb="md" ta={'left'}>Blog</Title>
      
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
