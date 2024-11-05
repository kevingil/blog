import http from '@/services/http';
import { PageResponse, PagesResponse, CreatePageData, UpdatePageData } from '@/services/pages/types';

class PageService {
  async getAllPages(params?: {
    active?: boolean;
    showInNav?: boolean;
  }): Promise<PagesResponse> {
    try {
      const queryParams = new URLSearchParams();
      if (params?.active !== undefined) {
        queryParams.append('active', params.active.toString());
      }
      if (params?.showInNav !== undefined) {
        queryParams.append('show_in_nav', params.showInNav.toString());
      }

      const response = await http.get(`/pages?${queryParams.toString()}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching pages:', error);
      throw error;
    }
  }

  async getPageBySlug(slug: string): Promise<PageResponse> {
    try {
      const response = await http.get(`/pages/${slug}`);
      return response.data;
    } catch (error) {
      console.error(`Error fetching page with slug ${slug}:`, error);
      throw error;
    }
  }

  async createPage(pageData: CreatePageData): Promise<PageResponse> {
    try {
      const response = await http.post('/pages', pageData);
      return response.data;
    } catch (error) {
      console.error('Error creating page:', error);
      throw error;
    }
  }

  async updatePage(pageData: UpdatePageData): Promise<PageResponse> {
    try {
      const response = await http.put(
        `/pages/${pageData.id}`,
        pageData
      );
      return response.data;
    } catch (error) {
      console.error(`Error updating page ${pageData.id}:`, error);
      throw error;
    }
  }

  async deletePage(id: number): Promise<{ success: boolean; message: string }> {
    try {
      const response = await http.delete(`/pages/${id}`);
      return response.data;
    } catch (error) {
      console.error(`Error deleting page ${id}:`, error);
      throw error;
    }
  }

  async getNavigationPages(): Promise<PagesResponse> {
    try {
      const response = await http.get('/pages/navigation');
      return response.data;
    } catch (error) {
      console.error('Error fetching navigation pages:', error);
      throw error;
    }
  }

  async togglePageStatus(id: number, isActive: boolean): Promise<PageResponse> {
    try {
      const response = await http.patch(`/pages/${id}/status`, {
        is_active: isActive
      });
      return response.data;
    } catch (error) {
      console.error(`Error toggling page status ${id}:`, error);
      throw error;
    }
  }

  async updatePageOrder(id: number, order: number): Promise<PageResponse> {
    try {
      const response = await http.patch(`/pages/${id}/order`, {
        nav_order: order
      });
      return response.data;
    } catch (error) {
      console.error(`Error updating page order ${id}:`, error);
      throw error;
    }
  }
}

export default new PageService();
