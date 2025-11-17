import { VITE_API_BASE_URL } from './constants';
import { handleApiResponse, getAuthHeaders } from './apiHelpers';
import type { ChatMessage } from './artifacts';

export interface ConversationHistory {
  messages: ChatMessage[];
  article_id: string;
  total: number;
}

// Get conversation history for an article
export async function getConversationHistory(
  articleId: string,
  limit: number = 50
): Promise<ConversationHistory> {
  const params = new URLSearchParams({
    limit: limit.toString(),
  });

  const response = await fetch(
    `${VITE_API_BASE_URL}/agent/conversations/${articleId}?${params}`,
    {
      headers: getAuthHeaders(),
    }
  );

  return handleApiResponse<ConversationHistory>(response);
}

// Get recent conversations (most recent messages across all articles)
export async function getRecentConversations(limit: number = 10): Promise<ChatMessage[]> {
  // This would require a new endpoint - for now, return empty
  // Can be implemented later as needed
  return [];
}

