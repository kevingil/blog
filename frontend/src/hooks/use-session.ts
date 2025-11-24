import { useState, useEffect, useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { getConversationHistory } from '@/services/conversations';
import { VITE_API_BASE_URL } from '@/services/constants';

// Types
export type SearchResult = {
  title: string;
  url: string;
  summary?: string;
  author?: string;
  published_date?: string;
  favicon?: string;
  highlights?: string[];
  text_preview?: string;
  has_full_text?: boolean;
};

export type SourceInfo = {
  source_id: string;
  original_title: string;
  original_url: string;
  content_length: number;
  source_type?: string;
  search_query?: string;
};

export type ChatMessage = {
  id?: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  diffState?: 'accepted' | 'rejected';
  diffPreview?: {
    oldText: string;
    newText: string;
    reason?: string;
  };
  meta_data?: {
    artifact?: {
      id: string;
      type: string;
      status: string;
      content: string;
      diff_preview?: string;
      title?: string;
      description?: string;
      applied_at?: string;
    };
    task_status?: any;
    tool_execution?: any;
    context?: any;
    user_action?: any;
  };
  tool_context?: {
    tool_name: string;
    tool_id: string;
    status: 'starting' | 'running' | 'completed' | 'error';
    search_query?: string;
    search_results?: SearchResult[];
    sources_created?: SourceInfo[];
    total_found?: number;
    sources_successful?: number;
    message?: string;
  };
  created_at?: string;
};

export interface SessionState {
  sessionId: string | null;
  articleId: string | null;
  messages: ChatMessage[];
  isLoading: boolean;
  isConnected: boolean;
  error: string | null;
}

export interface UseSessionOptions {
  articleId?: string;
  autoLoad?: boolean;
}

export interface UseSessionReturn extends SessionState {
  loadMessages: () => Promise<void>;
  addMessage: (message: ChatMessage) => void;
  updateMessage: (index: number, updates: Partial<ChatMessage>) => void;
  clearMessages: () => void;
  connectWebSocket: (requestId: string) => Promise<void>;
  disconnectWebSocket: () => void;
  sendMessage: (content: string, documentContent?: string) => Promise<string | null>;
}

/**
 * Centralized session management hook for chat/agent interactions
 * Handles message loading, WebSocket connections, and state management
 */
export function useSession(options: UseSessionOptions = {}): UseSessionReturn {
  const { articleId, autoLoad = true } = options;
  
  const queryClient = useQueryClient();
  const [state, setState] = useState<SessionState>({
    sessionId: articleId || null, // Article ID IS the session ID
    articleId: articleId || null,
    messages: [], // Backend handles initial greeting
    isLoading: false,
    isConnected: false,
    error: null,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();

  // Convert database messages to chat messages format
  const transformMessage = useCallback((msg: any): ChatMessage => {
    const chatMsg: ChatMessage = {
      id: msg.id,
      role: msg.role,
      content: msg.content,
      meta_data: msg.meta_data,
      created_at: msg.created_at,
    };
    
    // Reconstruct tool_context from metadata for search tools
    if (msg.meta_data?.tool_execution?.tool_name === 'search_web_sources') {
      const output = msg.meta_data.tool_execution.output;
      chatMsg.tool_context = {
        tool_name: 'search_web_sources',
        tool_id: msg.meta_data.tool_execution.tool_id || '',
        status: 'completed',
        search_query: output.query,
        search_results: output.search_results || [],
        sources_created: output.sources_created || [],
        total_found: output.total_found || 0,
        sources_successful: output.sources_successful || 0,
        message: output.message
      };
    }
    
    return chatMsg;
  }, []);

  // Load messages from database
  const loadMessages = useCallback(async () => {
    if (!articleId) {
      console.warn('[useSession] Cannot load messages: no articleId provided');
      return;
    }

    console.log(`[useSession] Loading messages for article: ${articleId}`);
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const result = await getConversationHistory(articleId);
      console.log(`[useSession] API returned:`, result);
      
      const transformedMessages = result.messages.map(transformMessage);
      
      setState(prev => ({
        ...prev,
        messages: transformedMessages, // Backend includes initial greeting if needed
        isLoading: false,
        articleId,
        sessionId: articleId, // Article ID IS the session ID
      }));

      console.log(`[useSession] ✅ Loaded ${transformedMessages.length} messages for article ${articleId}`);
      console.log(`[useSession] Messages:`, transformedMessages);
    } catch (error) {
      console.error('[useSession] ❌ Failed to load messages:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to load messages',
        isLoading: false,
      }));
    }
  }, [articleId, initialMessages, transformMessage]);

  // Auto-load messages when articleId changes
  useEffect(() => {
    if (autoLoad && articleId) {
      loadMessages();
    }
  }, [articleId, autoLoad, loadMessages]);

  // Add a new message
  const addMessage = useCallback((message: ChatMessage) => {
    setState(prev => ({
      ...prev,
      messages: [...prev.messages, message],
    }));
  }, []);

  // Update a message by index
  const updateMessage = useCallback((index: number, updates: Partial<ChatMessage>) => {
    setState(prev => ({
      ...prev,
      messages: prev.messages.map((msg, i) => 
        i === index ? { ...msg, ...updates } : msg
      ),
    }));
  }, []);

  // Clear all messages
  const clearMessages = useCallback(() => {
    setState(prev => ({
      ...prev,
      messages: [],
    }));
  }, []);

  // Connect to WebSocket for real-time updates
  const connectWebSocket = useCallback(async (requestId: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `${VITE_API_BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://')}/websocket`;
        console.log('[useSession] Connecting to WebSocket:', wsUrl);
        
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          console.log('[useSession] WebSocket connected, subscribing to:', requestId);
          ws.send(JSON.stringify({
            action: 'subscribe',
            requestId: requestId
          }));
          
          setState(prev => ({ ...prev, isConnected: true, sessionId: requestId }));
          resolve();
        };

        ws.onerror = (error) => {
          console.error('[useSession] WebSocket error:', error);
          setState(prev => ({ 
            ...prev, 
            isConnected: false,
            error: 'WebSocket connection error'
          }));
          reject(error);
        };

        ws.onclose = (event) => {
          console.log('[useSession] WebSocket closed:', event.code, event.reason);
          setState(prev => ({ ...prev, isConnected: false, sessionId: null }));
          wsRef.current = null;
          
          // Attempt to reconnect if unexpected closure
          if (event.code !== 1000) {
            reconnectTimeoutRef.current = setTimeout(() => {
              console.log('[useSession] Attempting to reconnect...');
              connectWebSocket(requestId).catch(console.error);
            }, 3000);
          }
        };
      } catch (error) {
        console.error('[useSession] Failed to create WebSocket:', error);
        reject(error);
      }
    });
  }, []);

  // Disconnect WebSocket
  const disconnectWebSocket = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log('[useSession] Disconnecting WebSocket');
      wsRef.current.close(1000, 'Client disconnect');
      wsRef.current = null;
    }
    
    setState(prev => ({ ...prev, isConnected: false, sessionId: null }));
  }, []);

  // Send a message (placeholder - to be implemented with actual API call)
  const sendMessage = useCallback(async (
    content: string, 
    documentContent?: string
  ): Promise<string | null> => {
    if (!articleId) {
      console.warn('[useSession] Cannot send message: no articleId');
      return null;
    }

    // This would call your actual API to submit the message
    // and return the requestId for WebSocket subscription
    console.log('[useSession] Send message called:', { content, articleId });
    
    // Add user message immediately
    addMessage({
      role: 'user',
      content,
      created_at: new Date().toISOString(),
    });

    // Return null for now - implement with actual API call
    return null;
  }, [articleId, addMessage]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

  return {
    ...state,
    loadMessages,
    addMessage,
    updateMessage,
    clearMessages,
    connectWebSocket,
    disconnectWebSocket,
    sendMessage,
  };
}

