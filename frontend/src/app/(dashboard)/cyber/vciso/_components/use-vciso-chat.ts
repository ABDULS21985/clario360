'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';

import { apiGet, apiPost } from '@/lib/api';
import { getAccessToken } from '@/lib/auth';
import { API_ENDPOINTS } from '@/lib/constants';
import type {
  VCISOChatResponse,
  VCISOConversationDetail,
  VCISOConversationListItem,
  VCISOConversationMessage,
  VCISOSuggestedAction,
  VCISOSuggestion,
} from '@/types/cyber';

// ── Constants ────────────────────────────────────────────────────────────────

const MAX_RECONNECT_ATTEMPTS = 8;
const BACKOFF_DELAYS = [1000, 2000, 4000, 8000, 16000, 30000, 60000];

function getBackoffDelay(attempt: number): number {
  return BACKOFF_DELAYS[Math.min(attempt, BACKOFF_DELAYS.length - 1)];
}

// ── Types ────────────────────────────────────────────────────────────────────

export type ConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'offline';

interface StreamingChunk {
  messageId: string;
  text: string;
  done: boolean;
}

interface SuggestionsResponse {
  suggestions: VCISOSuggestion[];
}

interface ConversationsEnvelope {
  data: VCISOConversationListItem[];
}

// ── Hook ─────────────────────────────────────────────────────────────────────

export function useVCISOChat() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);
  const intentionalCloseRef = useRef(false);
  const streamBufferRef = useRef<Map<string, string>>(new Map());

  const [conversationId, setConversationId] = useState<string | null>(null);
  const [messages, setMessages] = useState<VCISOConversationMessage[]>([]);
  const [suggestions, setSuggestions] = useState<VCISOSuggestion[]>([]);
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting');
  const [statusText, setStatusText] = useState<string | null>(null);
  const [isSending, setIsSending] = useState(false);

  // ── Conversations query ──────────────────────────────────────────────────

  const conversationsQuery = useQuery({
    queryKey: ['vciso-conversations'],
    queryFn: () => apiGet<ConversationsEnvelope>(API_ENDPOINTS.CYBER_VCISO_CONVERSATIONS),
    staleTime: 30_000,
  });

  // ── Suggestions query ────────────────────────────────────────────────────

  const suggestionsQuery = useQuery({
    queryKey: ['vciso-suggestions', conversationId],
    queryFn: () =>
      apiGet<SuggestionsResponse>(
        API_ENDPOINTS.CYBER_VCISO_SUGGESTIONS,
        conversationId ? { conversation_id: conversationId } : undefined,
      ),
    staleTime: 15_000,
  });

  useEffect(() => {
    if (suggestionsQuery.data?.suggestions) {
      setSuggestions(suggestionsQuery.data.suggestions);
    }
  }, [suggestionsQuery.data]);

  // ── WebSocket with reconnection ──────────────────────────────────────────

  const connect = useCallback(() => {
    const token = getAccessToken();
    if (!token) {
      setConnectionState('offline');
      return;
    }

    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    const wsProtocol = apiUrl.startsWith('https') ? 'wss' : 'ws';
    const wsBase = apiUrl.replace(/^https?/, wsProtocol);
    const socket = new WebSocket(`${wsBase}/ws/v1/cyber/vciso/chat?token=${token}`);
    wsRef.current = socket;

    setConnectionState(attemptRef.current === 0 ? 'connecting' : 'reconnecting');

    socket.onopen = () => {
      setConnectionState('connected');
      attemptRef.current = 0;
    };

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as Record<string, unknown>;
        handleWSMessage(payload);
      } catch {
        // Ignore malformed payloads
      }
    };

    socket.onclose = () => {
      wsRef.current = null;
      if (intentionalCloseRef.current) {
        return;
      }
      scheduleReconnect();
    };

    socket.onerror = () => {
      // onclose will fire after onerror, so reconnect is handled there
    };
  }, []);

  function scheduleReconnect() {
    if (attemptRef.current >= MAX_RECONNECT_ATTEMPTS) {
      setConnectionState('offline');
      return;
    }
    const delay = getBackoffDelay(attemptRef.current);
    attemptRef.current += 1;
    setConnectionState('reconnecting');
    reconnectTimerRef.current = setTimeout(() => connect(), delay);
  }

  function handleWSMessage(payload: Record<string, unknown>) {
    switch (payload.type) {
      case 'suggestions':
        setSuggestions(Array.isArray(payload.data) ? (payload.data as VCISOSuggestion[]) : []);
        break;

      case 'status':
        setStatusText(formatStatus(payload.status));
        break;

      case 'stream_start': {
        const msgId = String(payload.message_id ?? '');
        if (msgId) {
          streamBufferRef.current.set(msgId, '');
          setIsSending(true);
          setStatusText(null);
          setConversationId(String(payload.conversation_id));
          appendAssistantMessage({
            id: msgId,
            role: 'assistant',
            content: '',
            response_type: 'text',
            actions: [],
            created_at: new Date().toISOString(),
            _streaming: true,
          } as VCISOConversationMessage & { _streaming?: boolean });
        }
        break;
      }

      case 'stream_chunk': {
        const chunk = payload as unknown as StreamingChunk;
        const existing = streamBufferRef.current.get(chunk.messageId) ?? '';
        const updated = existing + chunk.text;
        streamBufferRef.current.set(chunk.messageId, updated);
        setMessages((current) =>
          current.map((m) => (m.id === chunk.messageId ? { ...m, content: updated } : m)),
        );
        break;
      }

      case 'stream_end': {
        const msgId = String(payload.message_id ?? '');
        streamBufferRef.current.delete(msgId);
        setMessages((current) =>
          current.map((m) =>
            m.id === msgId
              ? {
                  ...m,
                  content: String(payload.text ?? m.content),
                  response_type: String(payload.data_type ?? 'text') as VCISOConversationMessage['response_type'],
                  actions: Array.isArray(payload.actions) ? (payload.actions as VCISOSuggestedAction[]) : [],
                  tool_result: payload.data,
                  intent: payload.intent as string | undefined,
                }
              : m,
          ),
        );
        setIsSending(false);
        void conversationsQuery.refetch();
        break;
      }

      case 'response':
        setIsSending(false);
        setStatusText(null);
        setConversationId(String(payload.conversation_id));
        appendAssistantMessage({
          id: String(payload.message_id),
          role: 'assistant',
          content: String(payload.text ?? ''),
          response_type: String(payload.data_type ?? 'text') as VCISOConversationMessage['response_type'],
          actions: Array.isArray(payload.actions) ? (payload.actions as VCISOSuggestedAction[]) : [],
          tool_result: payload.data,
          created_at: new Date().toISOString(),
          intent: payload.intent as string | undefined,
        });
        void conversationsQuery.refetch();
        break;

      case 'error':
        setIsSending(false);
        setStatusText(null);
        toast.error(String(payload.message ?? 'Failed to process request'));
        break;

      default:
        break;
    }
  }

  useEffect(() => {
    connect();
    return () => {
      intentionalCloseRef.current = true;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmount');
        wsRef.current = null;
      }
    };
  }, [connect]);

  // ── Helpers ──────────────────────────────────────────────────────────────

  function appendAssistantMessage(message: VCISOConversationMessage) {
    setMessages((current) => [...current, message]);
  }

  function appendUserMessage(content: string) {
    setMessages((current) => [
      ...current,
      {
        id: `local-user-${Date.now()}`,
        role: 'user' as const,
        content,
        actions: [],
        created_at: new Date().toISOString(),
      },
    ]);
  }

  // ── Send message ─────────────────────────────────────────────────────────

  async function sendMessage(rawMessage: string) {
    const message = rawMessage.trim();
    if (!message || isSending) return;

    appendUserMessage(message);
    setIsSending(true);
    setStatusText(connectionState === 'connected' ? 'Classifying request...' : 'Sending request...');

    const socket = wsRef.current;
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'message', conversation_id: conversationId, content: message }));
      return;
    }

    // REST fallback when WebSocket is unavailable
    try {
      const response = await apiPost<VCISOChatResponse>(API_ENDPOINTS.CYBER_VCISO_CHAT, {
        conversation_id: conversationId,
        message,
      });
      setConversationId(response.conversation_id);
      appendAssistantMessage({
        id: response.message_id,
        role: 'assistant',
        content: response.response.text,
        response_type: response.response.data_type,
        actions: response.response.actions,
        tool_result: response.response.data,
        created_at: new Date().toISOString(),
        intent: response.intent,
      });
      const suggestionResponse = await apiGet<SuggestionsResponse>(API_ENDPOINTS.CYBER_VCISO_SUGGESTIONS, {
        conversation_id: response.conversation_id,
      });
      setSuggestions(suggestionResponse.suggestions ?? []);
      void conversationsQuery.refetch();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to process request');
    } finally {
      setIsSending(false);
      setStatusText(null);
    }
  }

  // ── Load conversation ────────────────────────────────────────────────────

  async function loadConversation(id: string) {
    try {
      const detail = await apiGet<VCISOConversationDetail>(`${API_ENDPOINTS.CYBER_VCISO_CONVERSATIONS}/${id}`);
      setConversationId(detail.id);
      setMessages(detail.messages);
      const suggestionResponse = await apiGet<SuggestionsResponse>(API_ENDPOINTS.CYBER_VCISO_SUGGESTIONS, {
        conversation_id: detail.id,
      });
      setSuggestions(suggestionResponse.suggestions ?? []);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to load conversation');
    }
  }

  // ── New chat ─────────────────────────────────────────────────────────────

  function startNewChat() {
    setConversationId(null);
    setMessages([]);
    setStatusText(null);
  }

  return {
    // State
    conversationId,
    messages,
    suggestions,
    connectionState,
    statusText,
    isSending,
    conversations: conversationsQuery.data?.data ?? [],

    // Actions
    sendMessage,
    loadConversation,
    startNewChat,
    setSuggestions,
  };
}

// ── Utilities ────────────────────────────────────────────────────────────────

function formatStatus(status: unknown): string {
  switch (status) {
    case 'classifying':
      return 'Classifying request...';
    case 'executing':
      return 'Executing tool...';
    case 'fetching':
      return 'Fetching data...';
    case 'building':
      return 'Building dashboard...';
    case 'investigating':
      return 'Analyzing alert...';
    case 'generating':
      return 'Generating report...';
    default:
      return 'vCISO is thinking...';
  }
}
