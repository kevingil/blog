'use server'

import ArticleEditor from '@/components/blog/Editor';
import { getUser } from '@/db/queries';
import { redirect } from 'next/navigation';

export default async function NewArticlePage() {
  const user = await getUser();
  if (!user) {
    redirect('/login');
  }

  return (
    <ArticleEditor isNew={true} />
  );
}
