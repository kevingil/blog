import { apiGet, apiDelete } from './authenticatedFetch';
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

  return apiGet<ConversationHistory>(
    `/agent/conversations/${articleId}?${params}`
  );
}

// Clear conversation history for an article
export async function clearConversationHistory(
  articleId: string
): Promise<{ success: boolean }> {
  return apiDelete<{ success: boolean }>(
    `/agent/conversations/${articleId}`
  );
}

// Get recent conversations (most recent messages across all articles)
export async function getRecentConversations(limit: number = 10): Promise<ChatMessage[]> {
  // This would require a new endpoint - for now, return empty
  // Can be implemented later as needed
  return [];
}

