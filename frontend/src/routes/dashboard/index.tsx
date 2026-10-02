import { useEffect, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import ArticleEditor from '@/components/blog/Editor';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/')({
  component: DashboardIndex,
});

function DashboardIndex() {
  const [session, setSession] = useState(0);
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    const reset = () => setSession((value) => value + 1);
    window.addEventListener("writing-session-reset", reset);
    return () => window.removeEventListener("writing-session-reset", reset);
  }, []);

  useEffect(() => {
    setPageTitle("New");
  }, [setPageTitle]);

  return (
    <section className="flex h-full min-h-0 flex-1 flex-col overflow-hidden">
      <ArticleEditor key={session} launchpad />
    </section>
  );
}
