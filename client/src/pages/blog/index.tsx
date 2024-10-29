import React, { useEffect } from 'react';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { Container, Title, Loader, Pagination, Group } from '@mantine/core';
import ArticlesList from '@/components/blog/ArticleList';
import { fetchArticles, setPage, setItemsPerPage } from '@/features/blog/blogSlice';
import { RootState } from '@/store/store';

const ArticlesPage: React.FC = () => {
  const dispatch = useAppDispatch();
  const { 
    articles, 
    loading,
    currentPage,
    totalPages,
    itemsPerPage,
    totalItems 
  } = useAppSelector((state: RootState) => state.blog);

  useEffect(() => {
    dispatch(fetchArticles({ page: currentPage, limit: itemsPerPage }));
  }, [dispatch, currentPage, itemsPerPage]);

  const handlePageChange = (newPage: number) => {
    dispatch(setPage(newPage));
  };

  const handleItemsPerPageChange = (value: string) => {
    dispatch(setItemsPerPage(Number(value)));
    dispatch(setPage(1)); // Reset to first page when changing items per page
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
