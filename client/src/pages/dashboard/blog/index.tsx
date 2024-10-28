import { useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  Paper,
  Title,
  Table,
  Menu,
  ActionIcon,
  Group,
  Text,
  Button,
  Box,
  LoadingOverlay
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconDots, IconPencil, IconTrash } from '@tabler/icons-react';
import { selectAuth } from '@/features/auth/authSlice';
import { fetchArticles, deleteArticle } from '@/features/blog/blogSlice';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { ITEMS_PER_PAGE } from '@/features/blog/types.d';

export default function ArticlesPage() {
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { isLoggedIn } = useAppSelector(selectAuth);
  const { loading, articles } = useAppSelector((state) => state.blog);

  useEffect(() => {
    if (!isLoggedIn) {
      navigate('/login', { replace: true });
      return;
    }

    dispatch(fetchArticles({ page: 1, limit: ITEMS_PER_PAGE }));
  }, [dispatch, isLoggedIn, navigate]);

  const handleDelete = async (id: number) => {
    try {
      await dispatch(deleteArticle(id)).unwrap();
      notifications.show({
        title: 'Success',
        message: 'Article deleted successfully',
        color: 'green',
      });
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: typeof error === 'string' ? error : 'Failed to delete article',
        color: 'red',
      });
    }
  };

  if (!isLoggedIn) {
    return null;
  }

  return (
    <Box p="md">
      <Group mb="lg">
        <Title order={2}>Articles</Title>
        <Button
          component={Link}
          to="/dashboard/blog/new"
          variant="filled"
        >
          New Article
        </Button>
      </Group>

      <Paper shadow="xs" p="md" pos="relative">
        <LoadingOverlay visible={loading} />
        <Table>
          <thead>
            <tr>
              <th>Title</th>
              <th>Author</th>
              <th>Tags</th>
              <th>Date</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {articles?.map((article) => (
              <tr key={article.article.id}>
                <td>
                  <Text
                    component={Link}
                    to={`/dashboard/blog/edit/${article.article.slug}`}
                  >
                    {article.article.title}
                  </Text>
                </td>
                <td>{article.author_name}</td>
                <td>
                  {article.tags
                    ? article.tags.map(tag => tag.tagName).join(', ')
                    : '-'
                  }
                </td>
                <td>
                  {new Date(article.article.createdAt).toLocaleDateString()}
                </td>
                <td>
                  <Menu position="bottom-end" withinPortal>
                    <Menu.Target>
                      <ActionIcon>
                        <IconDots size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item
                        component={Link}
                        to={`/dashboard/blog/edit/${article.article.slug}`}
                        leftSection={<IconPencil size={16} />}
                      >
                        Edit
                      </Menu.Item>
                      <Menu.Item
                        color="red"
                        leftSection={<IconTrash size={16} />}
                        onClick={() => handleDelete(article.article.id)}
                      >
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Paper>
    </Box>
  );
}
