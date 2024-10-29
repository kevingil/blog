import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { ArticleListItem } from './types';
import http from '@/services/http';

interface BlogListState {
  articles: ArticleListItem[];
  loading: boolean;
  error: string | null;
  currentPage: number;
  totalPages: number;
  totalItems: number;
  itemsPerPage: number;
}

const initialListState: BlogListState = {
  articles: [],
  loading: false,
  error: null,
  currentPage: 1,
  totalPages: 1,
  totalItems: 0,
  itemsPerPage: 10,
};

export const fetchArticlesList = createAsyncThunk(
  'blogList/fetchArticles',
  async ({ page = 1, limit = 10 }: { page: number; limit: number }, { rejectWithValue }) => {
    try {
      const response = await http.get(`/api/v1/blog/articles?page=${page}&limit=${limit}`);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to fetch articles');
    }
  }
);

export const deleteArticleFromList = createAsyncThunk(
  'blogList/deleteArticle',
  async (id: number, { rejectWithValue }) => {
    try {
      await http.delete(`/api/v1/blog/articles/${id}`);
      return id;
    } catch (error) {
      return rejectWithValue('Failed to delete article');
    }
  }
);

const blogListSlice = createSlice({
  name: 'blogList',
  initialState: initialListState,
  reducers: {
    setPage: (state, action) => {
      state.currentPage = action.payload;
    },
    setItemsPerPage: (state, action) => {
      state.itemsPerPage = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchArticlesList.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchArticlesList.fulfilled, (state, action) => {
        state.loading = false;
        state.articles = action.payload.articles;
        state.totalPages = action.payload.totalPages;
        state.totalItems = action.payload.totalItems;
        state.currentPage = action.payload.currentPage;
      })
      .addCase(fetchArticlesList.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })
      .addCase(deleteArticleFromList.fulfilled, (state, action) => {
        state.articles = state.articles.filter(
          (article) => article.id !== action.payload
        );
        state.totalItems -= 1;
      });
  },
});

export const { setPage, setItemsPerPage } = blogListSlice.actions;
export default blogListSlice.reducer;
