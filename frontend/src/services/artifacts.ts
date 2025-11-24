import { apiGet, apiPost } from './authenticatedFetch';

export interface ArtifactInfo {
  id: string;
  type: string;
  status: string;
  content: string;
  diff_preview?: string;
  title?: string;
  description?: string;
  applied_at?: string;
}

export interface ChatMessage {
  id: string;
  article_id: string;
  role: string;
  content: string;
  meta_data: {
    artifact?: ArtifactInfo;
    task_status?: any;
    tool_execution?: any;
    context?: any;
    user_action?: any;
  };
  created_at: string;
}

// Accept an artifact
export async function acceptArtifact(messageId: string, feedback?: string): Promise<{ status: string; message_id: string }> {
  return apiPost<{ status: string; message_id: string }>(
    `/agent/artifacts/${messageId}/accept`,
    { feedback: feedback || '' }
  );
}

// Reject an artifact
export async function rejectArtifact(messageId: string, reason?: string): Promise<{ status: string; message_id: string }> {
  return apiPost<{ status: string; message_id: string }>(
    `/agent/artifacts/${messageId}/reject`,
    { reason: reason || '' }
  );
}

// Get pending artifacts for an article
export async function getPendingArtifacts(articleId: string): Promise<{ artifacts: ChatMessage[]; total: number }> {
  return apiGet<{ artifacts: ChatMessage[]; total: number }>(
    `/agent/artifacts/${articleId}/pending`
  );
}

