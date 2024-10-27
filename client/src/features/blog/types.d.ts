
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
}
