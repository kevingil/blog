import { useEffect, useState } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { useForm } from '@mantine/form';
import { useDispatch, useSelector } from 'react-redux';
import { 
  Paper,
  Title,
  TextInput,
  Textarea,
  Button,
  Stack,
  Group,
  Box,
  LoadingOverlay
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { selectAuth } from '@/features/auth/authSlice';
import { updateArticle, fetchArticleData } from '@/features/blog/blogSlice';
import { AppDispatch } from '@/store/store';

interface ArticleFormData {
  id: number;
  title: string;
  content: string;
  image?: string;
  tags: string;
}

export default function EditArticlePage() {
  const navigate = useNavigate();
  const { slug } = useParams();
  const dispatch = useDispatch<AppDispatch>();
  const { isLoggedIn, user } = useSelector(selectAuth);
  const [isLoading, setIsLoading] = useState(false);
  
  const form = useForm<ArticleFormData>({
    initialValues: {
      id: 0,
      title: '',
      content: '',
      image: '',
      tags: '',
    },
    validate: {
      id: (value) => value ? 0 || null : 'ID is required', 
      title: (value) => value.length < 1 ? 'Title is required' : null,
      content: (value) => value.length < 1 ? 'Content is required' : null,
      image: (value) => value ? /^https?:\/\/.+/.test(value) ? null : 'Invalid URL' : null,
    },
  });

  useEffect(() => {
    if (!isLoggedIn) {
      navigate('/login', { replace: true });
      return;
    }

    async function fetchArticle() {
      if (slug) {
        setIsLoading(true);
        try {
          const response = await dispatch(fetchArticleData(slug)).unwrap();
          if (response) {
            form.setValues({
              title: response.article.title,
              content: response.article.content,
              image: response.article.image || '',
              tags: response.tags ? response.tags.map(tag => tag.tagName).join(', ') : '',
            });
          }
        } catch (error) {
          notifications.show({
            title: 'Error',
            message: 'Failed to fetch article',
            color: 'red',
          });
        } finally {
          setIsLoading(false);
        }
      }
    }
    fetchArticle();
  }, [slug, dispatch, isLoggedIn, navigate]);

  const handleSubmit = async (values: ArticleFormData) => {
    if (!isLoggedIn || !user.accessToken) {
      notifications.show({
        title: 'Error',
        message: 'You must be logged in to edit an article',
        color: 'red',
      });
      navigate('/login');
      return;
    }

    setIsLoading(true);
    try {
      await dispatch(updateArticle({
        id: parseInt(slug!),
        data: {
          article: {
            id:  Number(values.id),
            title: values.title,
            slug: values.title.toLowerCase().replace(/ /g, '-').replace(/[^\w-]+/g, ''),
            createdAt: new Date().toISOString(),
            content: values.content,
            image: values.image? values.image : null,
          },
          tags: values.tags.split(',').map(tag => ({ 
            tagId: undefined,
            tagName: tag.trim(),
            articleId: parseInt(slug!),
           })),
        },
      })).unwrap();

      notifications.show({
        title: 'Success',
        message: 'Article updated successfully',
        color: 'green',
      });
      navigate(`/blog/${slug}`);
    } catch (error) {
      console.error('Failed to update article:', error);
      notifications.show({
        title: 'Error',
        message: typeof error === 'string' ? error : 'Failed to update article. Please try again',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (!isLoggedIn) {
    return null; // Route protection will handle redirect
  }

  return (
    <Box p="md">
      <Title order={2} mb="lg">Edit Article</Title>
      
      <Paper shadow="xs" p="md" pos="relative">
        <LoadingOverlay visible={isLoading}  />
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            <TextInput
              label="Title"
              placeholder="Article Title"
              required
              {...form.getInputProps('title')}
            />

            <TextInput
              label="Image URL"
              placeholder="Optional, for header"
              {...form.getInputProps('image')}
            />

            <Textarea
              label="Content"
              placeholder="Article Content"
              minRows={10}
              required
              {...form.getInputProps('content')}
            />

            <TextInput
              label="Tags"
              placeholder="Tags (comma-separated)"
              {...form.getInputProps('tags')}
            />

            <Group mt="xl">
              <Button component={Link} to="/dashboard/blog" variant="subtle">
                Cancel
              </Button>
              <Button type="submit" loading={isLoading}>
                Update Article
              </Button>
            </Group>
          </Stack>
        </form>
      </Paper>
    </Box>
  );
}
