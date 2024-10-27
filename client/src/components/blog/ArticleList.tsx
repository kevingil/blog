import React from 'react';
import { Grid, Card, Image, Text, Group, Button, Badge } from '@mantine/core';
import { Link } from 'react-router-dom';
import { ArticleData } from '@/features/blog/types'; 

interface ArticlesListProps {
  articles: ArticleData[];
}

const ArticlesList: React.FC<ArticlesListProps> = ({ articles }) => {
  return (
    <Grid>
      {articles.map((articleData) => {
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
                <Text >{article.title}</Text>
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
      })}
    </Grid>
  );
};

export default ArticlesList;
