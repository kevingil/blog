import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import http from '@/services/http';

export interface FileData {
  key: string;
  size: string;
  lastModified: Date;
  url: string;
  isImage: boolean;
}

export interface FolderData {
  path: string;
  name: string;
  lastModified: Date;
}

interface StorageState {
  files: FileData[];
  folders: FolderData[];
  currentPath: string;
  loading: boolean;
  error: string | null;
}

const initialState: StorageState = {
  files: [],
  folders: [],
  currentPath: '',
  loading: false,
  error: null
};

export const fetchFiles = createAsyncThunk(
  'storage/fetchFiles',
  async (path: string) => {
    const response = await http.get(`/api/v1/storage?path=${encodeURIComponent(path)}`);
    return response.data;
  }
);

export const uploadFile = createAsyncThunk(
  'storage/uploadFile',
  async ({ path, file }: { path: string; file: File }) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', path);
    const response = await http.post('/api/v1/storage/upload', formData);
    return response.data;
  }
);

export const deleteFile = createAsyncThunk(
  'storage/deleteFile',
  async (key: string) => {
    await http.delete(`/api/v1/storage/${encodeURIComponent(key)}`);
    return key;
  }
);

export const createFolder = createAsyncThunk(
  'storage/createFolder',
  async (path: string) => {
    const response = await http.post('/api/v1/storage/folder', { path });
    return response.data;
  }
);

const storageSlice = createSlice({
  name: 'storage',
  initialState,
  reducers: {
    setCurrentPath: (state, action) => {
      state.currentPath = action.payload;
    }
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchFiles.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchFiles.fulfilled, (state, action) => {
        state.loading = false;
        state.files = action.payload.files;
        state.folders = action.payload.folders;
      })
      .addCase(fetchFiles.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch files';
      })
      .addCase(uploadFile.fulfilled, (state) => {
        state.loading = false;
      })
      .addCase(deleteFile.fulfilled, (state, action) => {
        state.files = state.files.filter(file => file.key !== action.payload);
      })
      .addCase(createFolder.fulfilled, (state) => {
        state.loading = false;
      });
  }
});

export const { setCurrentPath } = storageSlice.actions;
export default storageSlice.reducer;
