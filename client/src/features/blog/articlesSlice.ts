import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import axios from 'axios';

export interface Article {
  id: number;
  title: string;
  content: string;
  // Add other fields as necessary
}

interface ArticlesState {
  articles: Article[];
  loading: boolean;
}

const initialState: ArticlesState = {
  articles: [],
  loading: false,
};

// Define the fetchArticles async thunk
export const fetchArticles = createAsyncThunk<Article[]>(
  'articles/fetchArticles',
  async () => {
    const response = await axios.get('/api/articles'); // Update with your API endpoint
    return response.data;
  }
);

const articlesSlice = createSlice({
  name: 'articles',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchArticles.pending, (state) => {
        state.loading = true;
      })
      .addCase(fetchArticles.fulfilled, (state, action) => {
        state.articles = action.payload;
        state.loading = false;
      })
      .addCase(fetchArticles.rejected, (state) => {
        state.loading = false;
      });
  },
});

export default articlesSlice.reducer;
