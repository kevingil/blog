import { Insights } from '@/client';
import { generatedData } from './generatedClient';

// Types
export interface InsightTopic {
  id: string;
  organization_id?: string;
  name: string;
  description?: string;
  keywords: string[];
  is_auto_generated: boolean;
  content_count: number;
  last_insight_at?: string;
  color?: string;
  icon?: string;
  created_at: string;
  updated_at: string;
}

export interface Insight {
  id: string;
  organization_id?: string;
  topic_id?: string;
  title: string;
  summary: string;
  content?: string;
  key_points: string[];
  source_content_ids: string[];
  generated_at: string;
  period_start?: string;
  period_end?: string;
  is_read: boolean;
  is_pinned: boolean;
  is_used_in_article: boolean;
  meta_data?: Record<string, unknown>;
  topic_name?: string;
  topic_color?: string;
  topic_icon?: string;
}

export interface CrawledContent {
  id: string;
  data_source_id: string;
  url: string;
  title?: string;
  content: string;
  summary?: string;
  author?: string;
  published_at?: string;
  meta_data?: Record<string, unknown>;
  created_at: string;
  data_source_name?: string;
  data_source_url?: string;
}

export interface InsightWithSources extends Insight {
  source_contents: CrawledContent[];
  topic?: InsightTopic;
}

export interface CreateTopicRequest {
  name: string;
  description?: string;
  keywords?: string[];
  color?: string;
  icon?: string;
}

export interface UpdateTopicRequest {
  name?: string;
  description?: string;
  keywords?: string[];
  color?: string;
  icon?: string;
}

// Insight API calls

export interface ListInsightsResponse {
  insights: Insight[];
  total: number;
  page: number;
  limit: number;
}

export async function listInsights(
  page: number = 1, 
  limit: number = 20,
  topicId?: string
): Promise<ListInsightsResponse> {
  return generatedData<ListInsightsResponse>(
    Insights.listInsights({ query: { page, limit, topic_id: topicId } }),
  );
}

export async function getInsight(id: string): Promise<InsightWithSources> {
  return generatedData<InsightWithSources>(Insights.getInsight({ path: { id } }));
}

export async function markInsightAsRead(id: string): Promise<void> {
  await generatedData<{ success: boolean }>(Insights.markInsightRead({ path: { id } }));
}

export async function toggleInsightPinned(id: string): Promise<void> {
  await generatedData<{ success: boolean }>(Insights.toggleInsightPinned({ path: { id } }));
}

export async function deleteInsight(id: string): Promise<void> {
  await generatedData<{ success: boolean }>(Insights.deleteInsight({ path: { id } }));
}

export async function searchInsights(query: string, limit: number = 10): Promise<Insight[]> {
  return generatedData<Insight[]>(Insights.searchInsights({ query: { q: query, limit } }));
}

export async function getUnreadCount(): Promise<number> {
  const data = await generatedData<{ count: number }>(Insights.getUnreadInsightCount());
  return data.count;
}

// Topic API calls

export async function listTopics(): Promise<InsightTopic[]> {
  return generatedData<InsightTopic[]>(Insights.listInsightTopics());
}

export async function getTopic(id: string): Promise<InsightTopic> {
  return generatedData<InsightTopic>(Insights.getInsightTopic({ path: { id } }));
}

export async function createTopic(request: CreateTopicRequest): Promise<InsightTopic> {
  return generatedData<InsightTopic>(Insights.createInsightTopic({ body: request }));
}

export async function updateTopic(id: string, request: UpdateTopicRequest): Promise<InsightTopic> {
  return generatedData<InsightTopic>(
    Insights.updateInsightTopic({ path: { id }, body: request }),
  );
}

export async function deleteTopic(id: string): Promise<void> {
  await generatedData<{ success: boolean }>(
    Insights.deleteInsightTopic({ path: { id } }),
  );
}

// Crawled content API calls

export async function searchCrawledContent(query: string, limit: number = 10): Promise<CrawledContent[]> {
  return generatedData<CrawledContent[]>(
    Insights.searchInsightContent({ query: { q: query, limit } }),
  );
}

export async function getRecentCrawledContent(limit: number = 20): Promise<CrawledContent[]> {
  return generatedData<CrawledContent[]>(
    Insights.getRecentInsightContent({ query: { limit } }),
  );
}
