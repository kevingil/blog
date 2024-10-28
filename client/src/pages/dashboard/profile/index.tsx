import React, { useEffect } from 'react';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import {
  Container,
  Title,
  TextInput,
  Button,
  Group,
  Text,
  Image,
  Paper,
  Stack,
  Loader,
  FileButton,
  ActionIcon,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Trash } from 'lucide-react';
import { RootState } from '@/store/store';
import {
  fetchUserProfile,
  updateUserProfile,
  uploadAvatar,
  deleteAvatar,
  clearUpdateSuccess,
} from '@/features/profile/profileSlice';
import type { UpdateProfileData } from 'ProfileModels';

const ProfilePage: React.FC = () => {
  const dispatch = useAppDispatch();
  const { profile, loading, error, updateSuccess } = useAppSelector(
    (state: RootState) => state.profile
  );

  const form = useForm<UpdateProfileData>({
    initialValues: {
      name: '',
      email: '',
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value? value : '') ? null : 'Invalid email'),
      name: (value: string | undefined) => (value ? null : 'Name is required'),
    },
  });

  useEffect(() => {
    dispatch(fetchUserProfile());
  }, [dispatch]);

  useEffect(() => {
    if (profile) {
      form.setValues({
        name: profile.name,
        email: profile.email,
      });
    }
  }, [profile]);

  useEffect(() => {
    if (updateSuccess) {
      notifications.show({
        title: 'Success',
        message: 'Profile updated successfully',
        color: 'green',
      });
      dispatch(clearUpdateSuccess());
    }
  }, [updateSuccess, dispatch]);

  const handleSubmit = (values: UpdateProfileData) => {
    dispatch(updateUserProfile(values));
  };

  const handleAvatarUpload = (file: File | null) => {
    if (file) {
      dispatch(uploadAvatar(file));
    }
  };

  const handleAvatarDelete = () => {
    dispatch(deleteAvatar());
  };

  if (loading && !profile) {
    return (
      <Container size="sm">
        <Loader />
      </Container>
    );
  }

  return (
    <Container size="sm">
      <Title order={1} mb="xl">
        Profile Settings
      </Title>

      <Paper radius="md" p="xl" withBorder>
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            {/* Avatar Section */}
            <Group  >
              {profile?.avatar ? (
                <div className="relative">
                  <Image
                    src={profile.avatar}
                    width={120}
                    height={120}
                    radius={60}
                    alt="Profile"
                  />
                  <ActionIcon
                    color="red"
                    variant="filled"
                    className="absolute top-0 right-0"
                    onClick={handleAvatarDelete}
                  >
                    <Trash size={16} />
                  </ActionIcon>
                </div>
              ) : (
                <div className="w-[120px] h-[120px] bg-gray-200 rounded-full flex items-center justify-center">
                  <Text size="xl" color="dimmed">
                    No Avatar
                  </Text>
                </div>
              )}
              <FileButton onChange={handleAvatarUpload} accept="image/png,image/jpeg">
                {(props) => (
                  <Button variant="light" {...props}>
                    Upload Avatar
                  </Button>
                )}
              </FileButton>
            </Group>

            <TextInput
              required
              label="Name"
              placeholder="Your name"
              {...form.getInputProps('name')}
            />

            <TextInput
              required
              label="Email"
              placeholder="your.email@example.com"
              {...form.getInputProps('email')}
            />

            {profile?.role && (
              <Text size="sm" color="dimmed">
                Role: {profile.role.name}
              </Text>
            )}

            {error && (
              <Text color="red" size="sm">
                {error}
              </Text>
            )}

            <Group  mt="xl">
              <Button type="submit" loading={loading}>
                Save Changes
              </Button>
            </Group>
          </Stack>
        </form>
      </Paper>
    </Container>
  );
};

export default ProfilePage;
