export const ITEMS_PER_PAGE = 6;

export type ArticleListItem = {
  id: number;
  title: string | null;
  slug: string | null;
  createdAt: number;
  image: string | null;
  content: string | null;
  author: string | null;
  tags: (string | null)[];
};

export type Article = {
  id: number;
  title: string;
  slug: string;
  createdAt: string;
  image: string | null;
  content: string;
};


export type TagData = {
  articleId: number;
  tagId: number | undefined;
  tagName: string;
}

export type ArticleData = {
  article: Article;
  author_name: string;
  tags: TagData[] | null;
};

export type RecommendedArticle = {
  id: number;
  title: string;
  createdAt: string;
  image: string | null;
};

export interface BlogState {
  articleData: ArticleData | null;
  recommendedArticles: RecommendedArticle[] | null;
  loading: boolean;
}
