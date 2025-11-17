import { VITE_API_BASE_URL } from './constants';
import { handleApiResponse, getAuthHeaders } from './apiHelpers';

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
  const response = await fetch(`${VITE_API_BASE_URL}/agent/artifacts/${messageId}/accept`, {
    method: 'POST',
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ feedback: feedback || '' }),
  });

  return handleApiResponse<{ status: string; message_id: string }>(response);
}

// Reject an artifact
export async function rejectArtifact(messageId: string, reason?: string): Promise<{ status: string; message_id: string }> {
  const response = await fetch(`${VITE_API_BASE_URL}/agent/artifacts/${messageId}/reject`, {
    method: 'POST',
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ reason: reason || '' }),
  });

  return handleApiResponse<{ status: string; message_id: string }>(response);
}

// Get pending artifacts for an article
export async function getPendingArtifacts(articleId: string): Promise<{ artifacts: ChatMessage[]; total: number }> {
  const response = await fetch(`${VITE_API_BASE_URL}/agent/artifacts/${articleId}/pending`, {
    headers: getAuthHeaders(),
  });

  return handleApiResponse<{ artifacts: ChatMessage[]; total: number }>(response);
}

