import { Agent } from '@/client';
import { generatedData } from './generatedClient';
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
  return generatedData<ConversationHistory>(
    Agent.getConversationHistory({
      path: { articleId },
      query: { limit: String(limit) },
    }),
  );
}

// Clear conversation history for an article
export async function clearConversationHistory(
  articleId: string
): Promise<{ success: boolean }> {
  return generatedData<{ success: boolean }>(
    Agent.clearConversationHistory({ path: { articleId } }),
  );
}

// Get recent conversations (most recent messages across all articles)
export async function getRecentConversations(_limit: number = 10): Promise<ChatMessage[]> {
  // This would require a new endpoint - for now, return empty
  // Can be implemented later as needed
  return [];
}
