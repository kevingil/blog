import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import http from '@/services/http';

interface PaginationData {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  itemsPerPage: number;
}

interface BlogState extends PaginationData {
  articleData: ArticleData | null;
  recommendedArticles: RecommendedArticle[] | null;
  articles: ArticleData[];
  loading: boolean;
  error: string | null;
}

interface TagData {
  articleId: number;
  tagId: number;
  tagName: string;
}

interface ArticleData {
  article: {
    id: number;
    title: string;
    slug: string;
    content: string;
    image: string | null;
    createdAt: string;
  };
  tags: TagData[] | null;
  author_name: string;
}

interface RecommendedArticle {
  id: number;
  title: string;
  slug: string;
  image: string | null;
  createdAt: string;
  author: string | null;
}

const initialState: BlogState = {
  articleData: null,
  recommendedArticles: null,
  articles: [],
  loading: false,
  error: null,
  currentPage: 1,
  totalPages: 1,
  totalItems: 0,
  itemsPerPage: 10,
};

// Fetch paginated articles
export const fetchArticles = createAsyncThunk(
  'blog/fetchArticles',
  async ({ page = 1, limit = 10 }: { page: number; limit: number }, { rejectWithValue }) => {
    try {
      const response = await http.get(`/api/v1/blog/articles?page=${page}&limit=${limit}`);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to fetch articles');
    }
  }
);

// Create new article (requires auth)
export const createArticle = createAsyncThunk(
  'blog/createArticle',
  async (articleData: Partial<ArticleData>, { rejectWithValue }) => {
    try {
      const response = await http.post('/api/v1/blog/articles', articleData);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to create article');
    }
  }
);

// Update article (requires auth)
export const updateArticle = createAsyncThunk(
  'blog/updateArticle',
  async ({ id, data }: { id: number; data: Partial<ArticleData> }, { rejectWithValue }) => {
    try {
      const response = await http.put(`/api/v1/blog/articles/${id}`, data);
      return response.data;
    } catch (error) {
      return rejectWithValue('Failed to update article');
    }
  }
);

// Delete article (requires auth)
export const deleteArticle = createAsyncThunk(
  'blog/deleteArticle',
  async (id: number, { rejectWithValue }) => {
    try {
      await http.delete(`/api/v1/blog/articles/${id}`);
      return id;
    } catch (error) {
      return rejectWithValue('Failed to delete article');
    }
  }
);

// Keep existing thunks
export const fetchArticleData = createAsyncThunk(
  'blog/fetchArticleData',
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
  'blog/fetchRecommendedArticles',
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

const blogSlice = createSlice({
  name: 'blog',
  initialState,
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
      // Handle fetchArticles
      .addCase(fetchArticles.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchArticles.fulfilled, (state, action) => {
        state.loading = false;
        state.articles = action.payload.articles;
        state.totalPages = action.payload.totalPages;
        state.totalItems = action.payload.totalItems;
        state.currentPage = action.payload.currentPage;
      })
      .addCase(fetchArticles.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })

      // Handle createArticle
      .addCase(createArticle.fulfilled, (state, action) => {
        state.articles.unshift(action.payload);
        state.totalItems += 1;
      })

      // Handle updateArticle
      .addCase(updateArticle.fulfilled, (state, action) => {
        const index = state.articles.findIndex(
          (article) => article.article.id === action.payload.article.id
        );
        if (index !== -1) {
          state.articles[index] = action.payload;
        }
      })

      // Handle deleteArticle
      .addCase(deleteArticle.fulfilled, (state, action) => {
        state.articles = state.articles.filter(
          (article) => article.article.id !== action.payload
        );
        state.totalItems -= 1;
      })

      // Keep existing cases
      .addCase(fetchArticleData.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchArticleData.fulfilled, (state, action) => {
        state.loading = false;
        state.articleData = action.payload;
      })
      .addCase(fetchArticleData.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      })
      .addCase(fetchRecommendedArticles.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchRecommendedArticles.fulfilled, (state, action) => {
        state.loading = false;
        state.recommendedArticles = action.payload;
      })
      .addCase(fetchRecommendedArticles.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      });
  },
});

export const { setPage, setItemsPerPage } = blogSlice.actions;
export default blogSlice.reducer;
