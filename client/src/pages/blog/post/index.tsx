import React, { useEffect } from 'react';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { fetchArticleData, fetchRecommendedArticles } from '@/features/blog/blogSlice';
import { Badge, Card, Container, Divider, Group, Image, LoadingOverlay, Text, Title } from '@mantine/core';
import { useParams } from 'react-router-dom';
import { format } from 'date-fns';
import { RecommendedArticle, Tag, BlogState } from '@/features/blog/types'; 

const ArticlePage: React.FC = () => {
  const dispatch = useAppDispatch();
  const { slug } = useParams<{ slug: string }>();
  
  const { articleData, recommendedArticles, loading } = useAppSelector((state: BlogState) => state);

  useEffect(() => {
    if (slug) {
      dispatch(fetchArticleData(slug));
    }
  }, [dispatch, slug]);

  useEffect(() => {
    if (articleData) {
      dispatch(fetchRecommendedArticles(articleData.article.id));
    }
  }, [dispatch, articleData]);

  if (loading) {
    return <LoadingOverlay visible={loading} />;
  }

  if (!articleData) {
    return <Text>Article not found</Text>;
  }

  return (
    <Container size="lg" pt="md">
      <Title order={1}>{articleData.article.title}</Title>
      {articleData.article.image && (
        <Image
          src={articleData.article.image}
          alt={articleData.article.title}
          radius="md"
          height={400}
          fit="cover"
          mb="md"
        />
      )}
      <Group mb="sm">
        <Text>{articleData.author_name}</Text>
        <Text size="sm" color="dimmed">
          {format(new Date(articleData.article.createdAt), 'MMMM d, yyyy')}
        </Text>
      </Group>
      <Divider my="lg" />
      <Text dangerouslySetInnerHTML={{ __html: articleData.article.content }} />

      <Group mt="lg">
        {articleData.tags?.map((tag: Tag) => (
          <Badge key={tag.tagId} color="primary" variant="outline">
            {tag.tagName}
          </Badge>
        ))}
      </Group>

      <Divider my="xl" />

      <Title order={2} mb="md">
        Other Articles
      </Title>
      <RecommendedArticles recommendedArticles={recommendedArticles} />
    </Container>
  );
};

const RecommendedArticles: React.FC<{ recommendedArticles: RecommendedArticle[] | null }> = ({ recommendedArticles }) => {
  if (!recommendedArticles) {
    return <Text>No recommendations available</Text>;
  }

  return (
    <Group >
      {recommendedArticles.map((article) => (
        <Card key={article.id} shadow="sm" p="md" radius="md" withBorder>
          {article.image && (
            <Card.Section>
              <Image src={article.image} alt={article.title} height={160} />
            </Card.Section>
          )}
          <Title order={3} mt="sm" mb="xs">
            {article.title}
          </Title>
          <Text size="xs" color="dimmed" mb="sm">
            {format(new Date(article.createdAt), 'MMMM d, yyyy')}
          </Text>
        </Card>
      ))}
    </Group>
  );
};

export default ArticlePage;
