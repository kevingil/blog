import ArticleEditor from '@/components/blog/Editor';
import { useAuth } from '@/services/auth/auth';
import { redirect } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/dashboard/blog/new')({
  component: NewArticlePage,
});

function NewArticlePage() {
  const { user } = useAuth();

  if (!user) {
    redirect({ to: '/login' });
  }

  return (
    <ArticleEditor isNew={true} />
  );
}
