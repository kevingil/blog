import { Agent } from '@/client';
import { generatedData } from './generatedClient';

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
  await generatedData<{ success: boolean }>(
    Agent.acceptArtifact({
      path: { messageId },
      body: { feedback: feedback || '' },
    }),
  );
  return { status: 'accepted', message_id: messageId };
}

// Reject an artifact
export async function rejectArtifact(messageId: string, reason?: string): Promise<{ status: string; message_id: string }> {
  await generatedData<{ success: boolean }>(
    Agent.rejectArtifact({
      path: { messageId },
      body: { feedback: reason || '' },
    }),
  );
  return { status: 'rejected', message_id: messageId };
}

// Get pending artifacts for an article
export async function getPendingArtifacts(articleId: string): Promise<{ artifacts: ChatMessage[]; total: number }> {
  const data = await generatedData<{ artifacts: ChatMessage[] }>(
    Agent.getPendingArtifacts({ path: { articleId } }),
  );
  return { artifacts: data.artifacts, total: data.artifacts.length };
}
