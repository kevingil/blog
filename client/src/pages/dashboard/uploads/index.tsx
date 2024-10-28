import { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { IconFile, IconFolder, IconTrash, IconUpload } from '@tabler/icons-react';
import {
  AppShell,
  Container,
  Title,
  Paper,
  Group,
  Button,
  TextInput,
  Table,
  Modal,
  Stack,
  Text,
  FileButton,
  Image,
  Loader,
  ActionIcon,
  Breadcrumbs,
  Anchor,
  Box,
  CopyButton,
  Code
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import {
  fetchFiles,
  uploadFile,
  deleteFile,
  createFolder,
  setCurrentPath,
  FileData
} from '@/features/storage/storageSlice';

import type { RootState } from '@/store/store';


export default function UploadsPage() {
  const dispatch = useAppDispatch();
  const { files, folders, currentPath, loading } = useAppSelector((state: RootState) => state.storage);
  const [newFolderName, setNewFolderName] = useState('');
  const [selectedFile, setSelectedFile] = useState<FileData | null>(null);

  useEffect(() => {
    dispatch(fetchFiles(currentPath) as any);
  }, [currentPath, dispatch]);

  const handleFileUpload = async (file: File | null) => {
    if (file) {
      try {
        await dispatch(uploadFile({ path: `${currentPath}${file.name}`, file }) as any);
        dispatch(fetchFiles(currentPath) as any);
        notifications.show({
          title: 'Success',
          message: 'File uploaded successfully',
          color: 'green'
        });
      } catch (error) {
        notifications.show({
          title: 'Error',
          message: 'Failed to upload file',
          color: 'red'
        });
      }
    }
  };

  const handleDeleteFile = async (key: string) => {
    try {
      await dispatch(deleteFile(key) as any);
      notifications.show({
        title: 'Success',
        message: 'File deleted successfully',
        color: 'green'
      });
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to delete file',
        color: 'red'
      });
    }
  };

  const handleCreateFolder = async () => {
    if (newFolderName) {
      try {
        await dispatch(createFolder(`${currentPath}${newFolderName}/`) as any);
        setNewFolderName('');
        dispatch(fetchFiles(currentPath) as any);
        notifications.show({
          title: 'Success',
          message: 'Folder created successfully',
          color: 'green'
        });
      } catch (error) {
        notifications.show({
          title: 'Error',
          message: 'Failed to create folder',
          color: 'red'
        });
      }
    }
  };

  const navigateToFolder = (path: string) => {
    dispatch(setCurrentPath(path));
  };

  const navigateUp = () => {
    const newPath = currentPath.split('/').slice(0, -2).join('/') + '/';
    dispatch(setCurrentPath(newPath));
  };

  const formatMarkdownLink = (file: FileData) => {
    return file.isImage
      ? `![${file.key}](${file.url})`
      : `[${file.key}](${file.url})`;
  };

  const breadcrumbs = currentPath.split('/').filter(Boolean).map((part, index, array) => {
    const path = array.slice(0, index + 1).join('/') + '/';
    return (
      <Anchor key={path} onClick={() => navigateToFolder(path)}>
        {part}
      </Anchor>
    );
  });

  return (
    <AppShell>
      <Container size="xl" py="md">
        <Title order={2} mb="lg">File Storage</Title>

        <Paper shadow="xs" p="md" mb="md">
          <Group >
            <Group>
              <FileButton onChange={handleFileUpload} accept="*">
                {(props) => (
                  <Button {...props}>
                    <IconUpload size={16} />
                    Upload File
                  </Button>
                )}
              </FileButton>

              <Group>
                <TextInput
                  placeholder="New folder name"
                  value={newFolderName}
                  onChange={(e) => setNewFolderName(e.target.value)}
                />
                <Button onClick={handleCreateFolder}>Create Folder</Button>
              </Group>
            </Group>
          </Group>
        </Paper>

        <Paper shadow="xs" p="md">
          <Group mb="md">
            <Button
              variant="light"
              onClick={navigateUp}
              disabled={currentPath === ''}
            >
              Up
            </Button>
            <Breadcrumbs>{breadcrumbs}</Breadcrumbs>
          </Group>

          {loading ? (
            <Box ta="center" py="xl">
              <Loader />
            </Box>
          ) : (
            <Table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Size</th>
                  <th>Last Modified</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {folders.map((folder) => (
                  <tr key={folder.path}>
                    <td>
                      <Group onClick={() => navigateToFolder(folder.path)} style={{ cursor: 'pointer' }}>
                        <IconFolder size={16} />
                        {folder.name}
                      </Group>
                    </td>
                    <td>-</td>
                    <td>{folder.lastModified.toLocaleString()}</td>
                    <td>-</td>
                  </tr>
                ))}
                {files.map((file) => (
                  <tr key={file.key}>
                    <td>
                      <Group onClick={() => setSelectedFile(file)} style={{ cursor: 'pointer' }}>
                        {file.isImage ? (
                          <Image src={file.url} width={20} height={20} />
                        ) : (
                          <IconFile size={16} />
                        )}
                        {file.key.split('/').pop()}
                      </Group>
                    </td>
                    <td>{file.size}</td>
                    <td>{file.lastModified.toLocaleString()}</td>
                    <td>
                      <ActionIcon color="red" onClick={() => handleDeleteFile(file.key)}>
                        <IconTrash size={16} />
                      </ActionIcon>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </Paper>

        <Modal
          opened={!!selectedFile}
          onClose={() => setSelectedFile(null)}
          size="lg"
          title="File Details"
        >
          {selectedFile && (
            <Stack>
              {selectedFile.isImage && (
                <Image
                  src={selectedFile.url}
                  fit="contain"
                  height={300}
                />
              )}

              <Stack>
                <Text>File name</Text>
                <Text>{selectedFile.key}</Text>

                <Text>URL</Text>
                <Group>
                  <Text>{selectedFile.url}</Text>
                  <CopyButton value={selectedFile.url}>
                    {({ copied, copy }) => (
                      <Button color={copied ? 'teal' : 'blue'} onClick={copy}>
                        {copied ? 'Copied' : 'Copy'}
                      </Button>
                    )}
                  </CopyButton>
                </Group>

                <Text>Markdown</Text>
                <Group>
                  <Code block>{formatMarkdownLink(selectedFile)}</Code>
                  <CopyButton value={formatMarkdownLink(selectedFile)}>
                    {({ copied, copy }) => (
                      <Button color={copied ? 'teal' : 'blue'} onClick={copy}>
                        {copied ? 'Copied' : 'Copy'}
                      </Button>
                    )}
                  </CopyButton>
                </Group>

                <Text>Size</Text>
                <Text>{selectedFile.size}</Text>

                <Text>Last modified</Text>
                <Text>{new Date(selectedFile.lastModified).toLocaleString()}</Text>
              </Stack>

              <Group mt="xl">
                <Button
                  color="red"
                  onClick={() => {
                    handleDeleteFile(selectedFile.key);
                    setSelectedFile(null);
                  }}
                >
                  Delete
                </Button>
                <Button variant="light" onClick={() => setSelectedFile(null)}>
                  Close
                </Button>
              </Group>
            </Stack>
          )}
        </Modal>
      </Container>
    </AppShell>
  );
}
