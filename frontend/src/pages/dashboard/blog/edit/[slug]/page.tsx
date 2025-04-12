'use server'

import ArticleEditor from '@/components/blog/Editor';
import { getUser } from '@/db/queries';
import { redirect } from 'next/navigation';

export default async function EditArticlePage() {
  const user = await getUser();

  if (!user) {
    redirect('/login');
  }


  return (
    <ArticleEditor />
  );
}
