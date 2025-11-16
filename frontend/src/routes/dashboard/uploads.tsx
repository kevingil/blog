import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table';
import { listFiles, uploadFile, deleteFile, createFolder, FileData, FolderData } from '../../services/storage';
import { Folder, File, Trash2, Upload } from 'lucide-react';

import { VITE_PUBLIC_S3_URL_PREFIX } from '../../services/constants';

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose
} from "../../components/ui/dialog"
import { createFileRoute } from '@tanstack/react-router';
import { useAdminDashboard } from '@/services/dashboard/dashboard';

export const Route = createFileRoute('/dashboard/uploads')({
  component: UploadsPage,
});

function UploadsPage() {
  const [files, setFiles] = useState<FileData[]>([]);
  const [folders, setFolders] = useState<FolderData[]>([]);
  const [currentPath, setCurrentPath] = useState('');
  const [fileUpload, setFileUpload] = useState<File | null>(null);
  const [newFolderName, setNewFolderName] = useState('');
  const [fetchingError, setFetchingError] = useState<string | null>(null);
  const { setPageTitle } = useAdminDashboard();

  useEffect(() => {
    setPageTitle("Uploads");
  }, [setPageTitle]);

  console.log("VITE_PUBLIC_S3_URL_PREFIX", VITE_PUBLIC_S3_URL_PREFIX);
  
  const fetchFiles = async () => {
    try {
      const { files, folders } = await listFiles(currentPath);
      setFiles(files);
      setFolders(folders);
      setFetchingError(null);
    } catch (error: any) {
      const errorMessage = error?.message ?? 'An unknown error occurred';
      setFetchingError(errorMessage);
    }
  };

  useEffect(() => {
    fetchFiles();
  }, [currentPath]); 


  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setFileUpload(file);
      await uploadFile(`${currentPath}${file.name}`, file);
      setFileUpload(null);
      fetchFiles();
    }
  };

  const handleDeleteFile = async (key: string) => {
    await deleteFile(key);
    fetchFiles();
  };

  const handleCreateFolder = async () => {
    if (newFolderName) {
      await createFolder(`${currentPath}${newFolderName}/`);
      setNewFolderName('');
      fetchFiles();
    }
  };

  const navigateToFolder = (path: string) => {
    setCurrentPath(path);
  };

  const navigateUp = () => {
    const newPath = currentPath.split('/').slice(0, -2).join('/') + '/';
    setCurrentPath(newPath);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).catch(err => {
      console.error('Error copying to clipboard:', err);
    });
  };

  const formatMarkdownLink = (file: FileData | null) => {
    if (!file) {
      return '';
    } else {

      const markdownLink = file?.is_image
        ? `![${file.key}](${file.url})`
        : `[${file.key}](${file.url})`;
      return markdownLink
    }
  }


  return (
    <section className="flex-1 p-0 md:p-4">
      <div className='flex items-center gap-2 justify-between h-full'>
        <h1 className="text-lg lg:text-2xl font-medium text-gray-900 dark:text-white mb-6">
          Uploads
        </h1>
        {fetchingError && (
          <Card className='border-gray-200 dark:border-stone-700 p-2 text-gray-800 dark:text-white   mb-4'>
            <CardContent>
              <p className='text-gray-800 dark:text-white font-medium text-sm'><span className='font-medium text-red-500'>Error:</span> {fetchingError}</p>
            </CardContent>
          </Card>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>
              <div className='flex items-center gap-2 justify-between'>
                <p className='flex items-center gap-2'>
                    Upload File
                </p>
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center space-x-4">
              <Input type="file" onChange={handleFileUpload} />
              {fileUpload && <p>Uploading: {fileUpload.name}</p>}
            </div>
          </CardContent>
        </Card>

        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create Folder</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center space-x-4">
              <Input
                type="text"
                placeholder="New folder name"
                value={newFolderName}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewFolderName(e.target.value)}
              />
              <Button onClick={handleCreateFolder}>Create Folder</Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Files and Folders</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="mb-4">
            <Button onClick={navigateUp} disabled={currentPath === ''}
              className='bg-zinc-600/50 hover:bg-zinc-600/80'>Up</Button>
            <span className="ml-4">{currentPath || '/'}</span>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Size</TableHead>
                <TableHead>Last Modified</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {folders.map((folder) => (
                <TableRow key={folder.path}>
                  <TableCell>
                    <Button onClick={() => navigateToFolder(folder.path)}>
                      <Folder className="mr-2" />
                      {folder.name}
                    </Button>
                  </TableCell>
                  <TableCell>-</TableCell>
                  <TableCell>{folder?.lastModified?.toLocaleString()}</TableCell>
                  <TableCell>-</TableCell>
                </TableRow>
              ))}
              {files.map((file) => (
                <TableRow key={file.key}>
                  <TableCell>
                    <Dialog>
                      <DialogTrigger className='flex items-center text-left'>
                          {file.is_image ?
                          <img
                            src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
                            className="w-6 h-6 mr-2"
                          />
                          :
                          <File className="mr-2" />
                        }
                        {file.key.split('/').pop()}
                      </DialogTrigger>
                      <DialogContent className='w-full md:max-w-5xl max-h-[90vh] overflow-y-auto'>




                        <DialogHeader>
                          <DialogTitle className="text-xl font-medium">
                            File Detail
                          </DialogTitle>
                        </DialogHeader>

                        <div className="space-y-4">
                          {file?.is_image ? (
                            <div className="flex justify-center">
                              <img
                                src={`${VITE_PUBLIC_S3_URL_PREFIX}/${file.key}`}
                                alt={file.key}
                                className="max-h-[500px] p-4"
                              />
                            </div>
                          ) : (
                            <File className="w-full h-48 p-4 text-gray-600" />
                          )}

                          <div className="space-y-2">
                            <div>
                              <p className="text-sm font-medium">File name</p>
                              <p className="mt-1">{file?.key}</p>
                            </div>

                            <div>
                              <p className="text-sm font-medium">Link</p>
                              <div className="flex mt-1 gap-2">
                                <a href={file?.url} className="mt-1 break-all text-blue-600 hover:underline">
                                  {file?.url}
                                </a>

                                <Button
                                  className="px-3 rounded-l-none"
                                  onClick={() => copyToClipboard(file.url)}
                                >
                                  Copy
                                </Button>
                              </div>
                            </div>

                            <div>
                              <p className="text-sm font-medium">Markdown</p>
                              <div className="flex mt-1 gap-2">
                                <p className="flex-1 p-2 bg-gray-200 dark:bg-gray-800">
                                  {formatMarkdownLink(file)}
                                </p>
                                <Button
                                  className="px-3 rounded-l-none"
                                  onClick={() => copyToClipboard(formatMarkdownLink(file))}
                                >
                                  Copy
                                </Button>
                              </div>
                            </div>

                            <div>
                              <p className="text-sm font-medium">Size</p>
                              <p className="mt-1">{file?.size}</p>
                            </div>

                            <div>
                              <p className="text-sm font-medium">Last modified</p>
                              <p className="mt-1">
                                {file?.last_modified && new Date(file.last_modified).toLocaleString()}
                              </p>
                            </div>
                          </div>
                        </div>

                        <div className="mt-4 flex justify-between w-full gap-2">
                          <Button
                            onClick={() => handleDeleteFile(file.key)}
                            className="w-full sm:w-auto"
                          >
                            Delete
                          </Button>
                          <DialogClose asChild>
                            <Button type="button">
                              Close
                            </Button>
                          </DialogClose>
                        </div>

                      </DialogContent>
                    </Dialog>

                  </TableCell>
                  <TableCell>{file?.size}</TableCell>
                  <TableCell>{file?.last_modified?.toLocaleString()}</TableCell>
                  <TableCell>
                    <Button onClick={() => handleDeleteFile(file.key)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </section>
  );
}
