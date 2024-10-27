import React, { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Container, Title, Loader } from '@mantine/core';
import ArticlesList from '@/components/blog/ArticleList';
import { fetchArticles } from '@/features/blog/articlesSlice';
import { RootState } from '@/features/blog/store'; // Make sure to create a store

const ArticlesPage: React.FC = () => {
  const dispatch = useDispatch();
  const { articles, loading } = useSelector((state: RootState) => state.articles);

  useEffect(() => {
    dispatch(fetchArticles());
  }, [dispatch]);

  return (
    <Container>
      <Title order={1} mb="md">Blog</Title>
      {loading ? (
        <Loader />
      ) : (
        <ArticlesList articles={articles} pagination={true} />
      )}
    </Container>
  );
};

export default ArticlesPage;
