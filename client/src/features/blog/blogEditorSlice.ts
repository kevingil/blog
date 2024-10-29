import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { ArticleData } from './types';
import http from '@/services/http';

interface BlogEditorState {
  draftArticle: ArticleData | null;
  loading: boolean;
  error: string | null;
  saveStatus: 'idle' | 'saving' | 'saved' | 'error';
}

const initialEditorState: BlogEditorState = {
  draftArticle: null,
  loading: false,
  error: null,
  saveStatus: 'idle',
};

export const fetchArticleForEdit = createAsyncThunk(
  'blogEditor/fetchArticle',
  async (id: number, { rejectWithValue }) => {
    try {
      const response = await http.get<ArticleData>(`/api/v1/blog/articles/${id}/edit`);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to fetch article for editing');
    }
  }
);

export const saveArticle = createAsyncThunk(
  'blogEditor/saveArticle',
  async ({ id, data }: { id?: number; data: Partial<ArticleData> }, { rejectWithValue }) => {
    try {
      if (id) {
        const response = await http.put(`/api/v1/blog/articles/${id}`, data);
        return response.data;
      } else {
        const response = await http.post('/api/v1/blog/articles', data);
        return response.data;
      }
    } catch (error) {
      return rejectWithValue('Failed to save article');
    }
  }
);

const blogEditorSlice = createSlice({
  name: 'blogEditor',
  initialState: initialEditorState,
  reducers: {
    updateDraft: (state, action) => {
      state.draftArticle = {
        ...state.draftArticle,
        ...action.payload,
      };
      state.saveStatus = 'idle';
    },
    clearEditor: (state) => {
      state.draftArticle = null;
      state.saveStatus = 'idle';
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchArticleForEdit.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchArticleForEdit.fulfilled, (state, action) => {
        state.loading = false;
        state.draftArticle = action.payload;
      })
      .addCase(fetchArticleForEdit.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })
      .addCase(saveArticle.pending, (state) => {
        state.saveStatus = 'saving';
      })
      .addCase(saveArticle.fulfilled, (state, action) => {
        state.draftArticle = action.payload;
        state.saveStatus = 'saved';
      })
      .addCase(saveArticle.rejected, (state) => {
        state.saveStatus = 'error';
      });
  },
});

export const { updateDraft, clearEditor } = blogEditorSlice.actions;
export default blogEditorSlice.reducer;
