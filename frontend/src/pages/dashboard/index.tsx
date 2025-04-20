import { redirect } from 'next/navigation';
import { Settings } from './settings';
import { getUser } from '@/db/queries';

export default async function SettingsPage() {
  const user = await getUser();

  if (!user) {
    redirect('/login');
  }

  return <Settings/>;
}
