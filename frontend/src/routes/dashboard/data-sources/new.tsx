import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

export const Route = createFileRoute('/dashboard/data-sources/new')({
  component: NewDataSourcePage,
});

function NewDataSourcePage() {
  const navigate = useNavigate();

  useEffect(() => {
    // Redirect to main data sources page which has the create dialog
    navigate({ to: '/dashboard/data-sources' });
  }, [navigate]);

  return null;
}
