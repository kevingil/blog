import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { ArticleData, RecommendedArticle, TagData } from './types';
import http from '@/services/http';

interface BlogDetailState {
    articleData: ArticleData | null;
    recommendedArticles: RecommendedArticle[] | null;
    loading: boolean;
    error: string | null;
  }
  
  const initialDetailState: BlogDetailState = {
    articleData: null,
    recommendedArticles: null,
    loading: false,
    error: null,
  };
  
  export const fetchArticleDetail = createAsyncThunk(
    'blogDetail/fetchArticle',
    async (slug: string, { rejectWithValue }) => {
      try {
        const response = await http.get<ArticleData>(`/api/v1/blog/article/${slug}`);
        return response.data;
      } catch (error) {
        return rejectWithValue('Failed to fetch article');
      }
    }
  );
  
  export const fetchRecommendedArticles = createAsyncThunk(
    'blogDetail/fetchRecommended',
    async (currentArticleId: number, { rejectWithValue }) => {
      try {
        const response = await http.get<RecommendedArticle[]>(
          `/api/v1/blog/recommended/${currentArticleId}`
        );
        return response.data;
      } catch (error) {
        return rejectWithValue('Failed to fetch recommended articles');
      }
    }
  );
  
  const blogDetailSlice = createSlice({
    name: 'blogDetail',
    initialState: initialDetailState,
    reducers: {
      clearArticleDetail: (state) => {
        state.articleData = null;
        state.recommendedArticles = null;
      },
    },
    extraReducers: (builder) => {
      builder
        .addCase(fetchArticleDetail.pending, (state) => {
          state.loading = true;
          state.error = null;
        })
        .addCase(fetchArticleDetail.fulfilled, (state, action) => {
          state.loading = false;
          state.articleData = action.payload;
        })
        .addCase(fetchArticleDetail.rejected, (state, action) => {
          state.loading = false;
          state.error = action.payload as string;
        })
        .addCase(fetchRecommendedArticles.fulfilled, (state, action) => {
          state.recommendedArticles = action.payload;
        });
    },
  });
  
  export const { clearArticleDetail } = blogDetailSlice.actions;
  export default blogDetailSlice.reducer;
  