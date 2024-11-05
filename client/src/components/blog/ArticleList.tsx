import React from 'react';
import { Grid, Card, Image, Text, Group, Button, Badge, Skeleton } from '@mantine/core';
import { Link } from 'react-router-dom';
import { ArticleData } from '@/services/blog/types';

interface ArticlesListProps {
  articles: ArticleData[];
  loading?: boolean;
  pagination?: boolean;
  skeletonCount?: number;
}

const ArticlesList: React.FC<ArticlesListProps> = ({ 
  articles, 
  loading = false, 
  skeletonCount = 3 
}) => {
  const renderSkeleton = () => {
    return Array.from({ length: skeletonCount }).map((_, index) => (
      <Grid.Col key={`skeleton-${index}`}>
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          {/* Image skeleton */}
          <Card.Section>
            <Skeleton height={160} animate={true} />
          </Card.Section>

          {/* Title skeleton */}
          <Group mt="md" mb="xs">
            <Skeleton height={28} width="70%" animate={true} />
          </Group>

          {/* Content skeleton - 3 lines */}
          <Skeleton height={16} mt={6} width="90%" animate={true} />
          <Skeleton height={16} mt={6} width="85%" animate={true} />
          <Skeleton height={16} mt={6} width="80%" animate={true} />

          {/* Tags skeleton */}
          <Group mt="md">
            <Skeleton height={20} width={60} radius="xl" animate={true} />
            <Skeleton height={20} width={80} radius="xl" animate={true} />
            <Skeleton height={20} width={70} radius="xl" animate={true} />
          </Group>

          {/* Author skeleton */}
          <Skeleton height={16} mt="md" width={120} animate={true} />

          {/* Button skeleton */}
          <Skeleton height={36} mt="md" width="100%" radius="md" animate={true} />
        </Card>
      </Grid.Col>
    ));
  };

  const renderArticles = () => {
    return articles.map((articleData) => {
      const { article, author_name, tags } = articleData;
      return (
        <Grid.Col key={article.id}>
          <Card shadow="sm" padding="lg" radius="md" withBorder>
            {article.image && (
              <Card.Section>
                <Image
                  src={article.image}
                  height={160}
                  alt={article.title}
                />
              </Card.Section>
            )}
            <Group mt="md" mb="xs">
              <Text>{article.title}</Text>
            </Group>
            <Text size="sm" color="dimmed" lineClamp={3}>
              {article.content}
            </Text>
            <Group mt="md">
              {tags?.map((tag) => (
                <Badge key={tag.tagId} variant="light">
                  {tag.tagName}
                </Badge>
              ))}
            </Group>
            <Text size="sm" color="dimmed" mt="md">
              By {author_name}
            </Text>
            <Button
              component={Link}
              to={`/blog/${article.slug}`}
              variant="light"
              color="blue"
              fullWidth
              mt="md"
              radius="md"
            >
              Read more
            </Button>
          </Card>
        </Grid.Col>
      );
    });
  };

  return (
    <Grid >
      {loading ? renderSkeleton() : renderArticles()}
    </Grid>
  );
};

export default ArticlesList;
