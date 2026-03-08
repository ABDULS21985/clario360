'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Loader2, Wifi, WifiOff } from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { apiGet, apiPost } from '@/lib/api';
import { getAccessToken } from '@/lib/auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDateTime } from '@/lib/utils';
import type {
  VCISOChatResponse,
  VCISOConversationDetail,
  VCISOConversationListItem,
  VCISOConversationMessage,
  VCISOSuggestedAction,
  VCISOSuggestion,
} from '@/types/cyber';
import { ConversationList } from './conversation-list';
import { MessageBubble } from './message-bubble';
import { MessageInput } from './message-input';
import { SuggestionChips } from './suggestion-chips';

interface SuggestionsResponse {
  suggestions: VCISOSuggestion[];
}

interface ConversationsEnvelope {
  data: VCISOConversationListItem[];
}

type ConnectionState = 'connecting' | 'connected' | 'offline';

export function ChatPanel() {
  const router = useRouter();
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [messages, setMessages] = useState<VCISOConversationMessage[]>([]);
  const [input, setInput] = useState('');
  const [suggestions, setSuggestions] = useState<VCISOSuggestion[]>([]);
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting');
  const [statusText, setStatusText] = useState<string | null>(null);
  const [isSending, setIsSending] = useState(false);

  const conversationsQuery = useQuery({
    queryKey: ['vciso-conversations'],
    queryFn: () => apiGet<ConversationsEnvelope>(API_ENDPOINTS.CYBER_VCISO_CONVERSATIONS),
    staleTime: 30000,
  });

  useQuery({
    queryKey: ['vciso-suggestions', conversationId],
    queryFn: () =>
      apiGet<SuggestionsResponse>(API_ENDPOINTS.CYBER_VCISO_SUGGESTIONS, conversationId ? { conversation_id: conversationId } : undefined),
    onSuccess: (data) => setSuggestions(data.suggestions ?? []),
    staleTime: 15000,
  });

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, statusText]);

  useEffect(() => {
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

    socket.onopen = () => {
      setConnectionState('connected');
    };

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as Record<string, unknown>;
        switch (payload.type) {
          case 'suggestions':
            setSuggestions(Array.isArray(payload.data) ? (payload.data as VCISOSuggestion[]) : []);
            break;
          case 'status':
            setStatusText(formatStatus(payload.status));
            break;
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
      } catch {
        setConnectionState('offline');
      }
    };

    socket.onclose = () => {
      setConnectionState('offline');
    };

    socket.onerror = () => {
      setConnectionState('offline');
    };

    return () => {
      socket.close();
      wsRef.current = null;
    };
  }, [conversationsQuery]);

  function appendAssistantMessage(message: VCISOConversationMessage) {
    setMessages((current) => [...current, message]);
  }

  function appendUserMessage(content: string) {
    setMessages((current) => [
      ...current,
      {
        id: `local-user-${Date.now()}`,
        role: 'user',
        content,
        actions: [],
        created_at: new Date().toISOString(),
      },
    ]);
  }

  async function sendMessage(rawMessage?: string) {
    const message = (rawMessage ?? input).trim();
    if (!message || isSending) {
      return;
    }

    appendUserMessage(message);
    setInput('');
    setIsSending(true);
    setStatusText(connectionState === 'connected' ? 'Classifying request...' : 'Sending request...');

    const socket = wsRef.current;
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'message', conversation_id: conversationId, content: message }));
      return;
    }

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

  function handleAction(action: VCISOSuggestedAction) {
    switch (action.type) {
      case 'navigate':
        if (action.params.url) {
          router.push(action.params.url);
        }
        return;
      case 'confirm':
        if (window.confirm(action.params.warning ?? 'Do you want to continue?')) {
          void sendMessage(action.params.message ?? action.label);
        }
        return;
      case 'execute_tool':
      default:
        void sendMessage(action.params.message ?? action.label);
    }
  }

  function startNewChat() {
    setConversationId(null);
    setMessages([]);
    setStatusText(null);
  }

  return (
    <div className="flex h-[calc(100vh-12rem)] min-h-[720px] flex-col overflow-hidden rounded-[2rem] border bg-[linear-gradient(180deg,rgba(15,23,42,0.03),rgba(15,23,42,0))] shadow-xl">
      <div className="border-b bg-white/90 px-4 py-4 backdrop-blur">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="rounded-full bg-slate-900 text-white">
                Virtual CISO
              </Badge>
              <Badge variant="outline" className="rounded-full">
                {connectionState === 'connected' ? (
                  <>
                    <Wifi className="mr-1 h-3 w-3" />
                    Live
                  </>
                ) : (
                  <>
                    <WifiOff className="mr-1 h-3 w-3" />
                    Fallback
                  </>
                )}
              </Badge>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Deterministic security assistant with audited tool execution.
            </p>
          </div>
          <ConversationList
            conversations={conversationsQuery.data?.data ?? []}
            currentConversationId={conversationId}
            onNewChat={startNewChat}
            onSelect={(id) => void loadConversation(id)}
          />
        </div>
      </div>

      <SuggestionChips suggestions={suggestions} disabled={isSending} onSelect={(message) => void sendMessage(message)} />

      <ScrollArea className="flex-1 px-4 py-3">
        <div className="space-y-4">
          {messages.length === 0 ? (
            <div className="rounded-[1.75rem] border border-dashed bg-white/80 p-6">
              <p className="text-sm font-medium">Start with a direct question.</p>
              <p className="mt-2 text-sm text-muted-foreground">
                Try “What is our risk score?”, “Show critical alerts”, or “Build a dashboard for alerts and risk this week”.
              </p>
              <p className="mt-4 text-xs text-muted-foreground">Connected: {formatDateTime(new Date().toISOString())}</p>
            </div>
          ) : (
            messages.map((message) => (
              <MessageBubble key={message.id} message={message} onAction={handleAction} />
            ))
          )}
          {(isSending || statusText) && (
            <div className="flex items-center gap-3 rounded-2xl border bg-white px-4 py-3 text-sm text-muted-foreground shadow-sm">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{statusText ?? 'vCISO is thinking...'}</span>
            </div>
          )}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>

      <div className="border-t bg-slate-50/70 px-4 py-2 text-xs text-muted-foreground">
        {conversationId ? `Conversation ${conversationId.slice(0, 8)} active` : 'New conversation'}
      </div>

      <MessageInput value={input} onChange={setInput} onSend={() => void sendMessage()} disabled={isSending} />
    </div>
  );
}

function formatStatus(status: unknown): string {
  switch (status) {
    case 'classifying':
      return 'Classifying request...';
    case 'executing':
      return 'Executing tool...';
    default:
      return 'vCISO is thinking...';
  }
}
