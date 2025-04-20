'use client'

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Lock, Trash2, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { updatePassword, deleteAccount } from '@/services/auth/auth';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

type ActionState = {
  error?: string;
  success?: string;
};

export const Route = createFileRoute('/dashboard/security')({
  component: SecurityPage,
});

async function SecurityPage() {
  const navigate = useNavigate();
  const [passwordState, setPasswordState] = useState<ActionState>({ error: '', success: '' });
  const [deleteState, setDeleteState] = useState<ActionState>({ error: '', success: '' });
  const [isPasswordPending, setIsPasswordPending] = useState(false);
  const [isDeletePending, setIsDeletePending] = useState(false);

  const handlePasswordSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsPasswordPending(true);
    const formData = new FormData(event.currentTarget);
    
    try {
      await updatePassword(formData);
      setPasswordState({ success: 'Password updated successfully' });
    } catch (error) {
      setPasswordState({ error: 'Failed to update password' });
    } finally {
      setIsPasswordPending(false);
    }
  };

  const handleDeleteSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsDeletePending(true);
    const formData = new FormData(event.currentTarget);
    
    try {
      await deleteAccount(formData);
      setDeleteState({ success: 'Account deleted successfully' });
      navigate({ to: '/' });
    } catch (error) {
      setDeleteState({ error: 'Failed to delete account' });
    } finally {
      setIsDeletePending(false);
    }
  };

  return (
    <section className="flex-1 p-0 md:p-4">
      <h1 className="text-lg lg:text-2xl font-medium bold text-gray-900 dark:text-white mb-6">
        Security Settings
      </h1>
      <Card className="mb-8">
        <CardHeader>
          <CardTitle>Password</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={handlePasswordSubmit}>
            <div>
              <Label htmlFor="current-password">Current Password</Label>
              <Input
                id="current-password"
                name="currentPassword"
                type="password"
                autoComplete="current-password"
                required
                minLength={8}
                maxLength={100}
              />
            </div>
            <div>
              <Label htmlFor="new-password">New Password</Label>
              <Input
                id="new-password"
                name="newPassword"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                maxLength={100}
              />
            </div>
            <div>
              <Label htmlFor="confirm-password">Confirm New Password</Label>
              <Input
                id="confirm-password"
                name="confirmPassword"
                type="password"
                required
                minLength={8}
                maxLength={100}
              />
            </div>
            {passwordState.error && (
              <p className="text-red-500 text-sm">{passwordState.error}</p>
            )}
            {passwordState.success && (
              <p className="text-green-500 text-sm">{passwordState.success}</p>
            )}
            <Button
              type="submit"
              className="bg-primary hover:bg-primary/80 text-white"
              disabled={isPasswordPending}
            >
              {isPasswordPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Updating...
                </>
              ) : (
                <>
                  <Lock className="mr-2 h-4 w-4" />
                  Update Password
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Delete Account</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-gray-900 dark:text-white mb-4">
            Account deletion is non-reversable. Please proceed with caution.
          </p>
          <form onSubmit={handleDeleteSubmit} className="space-y-4">
            <div>
              <Label htmlFor="delete-password">Confirm Password</Label>
              <Input
                id="delete-password"
                name="password"
                type="password"
                required
                minLength={8}
                maxLength={100}
              />
            </div>
            {deleteState.error && (
              <p className="text-red-500 text-sm">{deleteState.error}</p>
            )}
            <Button
              type="submit"
              variant="destructive"
              className="bg-red-600 hover:bg-red-700"
              disabled={isDeletePending}
            >
              {isDeletePending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Deleting...
                </>
              ) : (
                <>
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Account
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </section>
  );
}
