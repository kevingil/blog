import ArticleEditor from '@/components/blog/Editor';
import { useAuth } from '@/services/auth/auth';
import { redirect } from '@tanstack/react-router';
import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/blog/new')({
  component: NewArticlePage,
});

function NewArticlePage() {
  const { user } = useAuth();
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("New Article");
  }, [setPageTitle]);

  if (!user) {
    redirect({ to: '/login' });
  }

  return (
    <ArticleEditor isNew={true} />
  );
}
