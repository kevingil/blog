import http from '@/services/http';
import { ArticleListItem, ArticleData, RecommendedArticle, ITEMS_PER_PAGE } from '@/services/blog/types';

interface FetchArticlesOptions {
  page?: number;
  tag?: string;
  search?: string;
}

interface ArticleResponse {
  articles: ArticleData[];
  total: number;
}

class BlogService {
  async fetchArticles({
    page = 1,
    tag,
    search
  }: FetchArticlesOptions = {}): Promise<ArticleResponse> {
    try {
      const queryParams = new URLSearchParams({
        page: page.toString(),
        limit: ITEMS_PER_PAGE.toString(),
        ...(tag && { tag }),
        ...(search && { search })
      });

      const response = await http.get(`/blog/articles?${queryParams}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching articles:', error);
      throw error;
    }
  }

  async fetchArticleBySlug(slug: string): Promise<ArticleData> {
    try {
      const response = await http.get(`/blog/articles/${slug}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching article:', error);
      throw error;
    }
  }

  async fetchRecommendedArticles(
    currentArticleId: number,
    limit: number = 3
  ): Promise<RecommendedArticle[]> {
    try {
      const queryParams = new URLSearchParams({
        articleId: currentArticleId.toString(),
        limit: limit.toString()
      });

      const response = await http.get(
        `/blog/articles/recommended?${queryParams}`
      );
      return response.data;
    } catch (error) {
      console.error('Error fetching recommended articles:', error);
      throw error;
    }
  }

  async fetchArticlesByAuthor(
    authorId: number,
    page: number = 1
  ): Promise<ArticleResponse> {
    try {
      const queryParams = new URLSearchParams({
        page: page.toString(),
        limit: ITEMS_PER_PAGE.toString()
      });

      const response = await http.get(
        `/blog/authors/${authorId}/articles?${queryParams}`
      );
      return response.data;
    } catch (error) {
      console.error('Error fetching author articles:', error);
      throw error;
    }
  }

  // Helper function to format the article data
  formatArticleData(data: ArticleData): ArticleData {
    return {
      ...data,
      article: {
        ...data.article,
        createdAt: new Date(data.article.createdAt).toISOString(),
      }
    };
  }
}

export default new BlogService();
