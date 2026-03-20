'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Loader2, Wifi, WifiOff, RefreshCcw } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { formatDateTime } from '@/lib/utils';
import type { VCISOSuggestedAction } from '@/types/cyber';
import { ConversationList } from './conversation-list';
import { MessageBubble } from './message-bubble';
import { MessageInput } from './message-input';
import { SuggestionChips } from './suggestion-chips';
import { useVCISOChat } from './use-vciso-chat';

export function ChatPanel() {
  const router = useRouter();
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const [input, setInput] = useState('');
  const [confirmAction, setConfirmAction] = useState<VCISOSuggestedAction | null>(null);

  const {
    conversationId,
    messages,
    suggestions,
    connectionState,
    statusText,
    isSending,
    conversations,
    preferredEngine,
    sendMessage,
    loadConversation,
    startNewChat,
    setPreferredEngine,
  } = useVCISOChat();

  // Auto-scroll on new messages or status changes
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, statusText]);

  function handleAction(action: VCISOSuggestedAction) {
    switch (action.type) {
      case 'navigate':
        if (action.params.url) {
          router.push(action.params.url);
        }
        return;
      case 'confirm':
        setConfirmAction(action);
        return;
      case 'execute_tool':
      default:
        void sendMessage(action.params.message ?? action.label);
    }
  }

  function handleConfirmAction() {
    if (confirmAction) {
      void sendMessage(confirmAction.params.message ?? confirmAction.label);
    }
  }

  const connectionBadge = (() => {
    switch (connectionState) {
      case 'connected':
        return (
          <Badge variant="outline" className="rounded-full">
            <Wifi className="mr-1 h-3 w-3" />
            Live
          </Badge>
        );
      case 'reconnecting':
        return (
          <Badge variant="outline" className="rounded-full text-amber-600 border-amber-300">
            <RefreshCcw className="mr-1 h-3 w-3 animate-spin" />
            Reconnecting
          </Badge>
        );
      default:
        return (
          <Badge variant="outline" className="rounded-full">
            <WifiOff className="mr-1 h-3 w-3" />
            Fallback
          </Badge>
        );
    }
  })();

  return (
    <>
      <div className="flex h-[calc(100vh-12rem)] min-h-[720px] flex-col overflow-hidden rounded-[2rem] border bg-[linear-gradient(180deg,rgba(15,23,42,0.03),rgba(15,23,42,0))] shadow-xl">
        {/* Header */}
        <div className="border-b bg-white/90 px-4 py-4 backdrop-blur">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="flex items-center gap-2">
                <Badge variant="secondary" className="rounded-full bg-slate-900 text-white">
                  Virtual CISO
                </Badge>
                {connectionBadge}
                <Badge variant="outline" className="rounded-full text-[10px] uppercase tracking-[0.12em]">
                  {preferredEngine === 'auto'
                    ? 'Auto route'
                    : preferredEngine === 'llm'
                      ? 'LLM forced'
                      : 'Deterministic forced'}
                </Badge>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">
                Hybrid vCISO assistant with transparent routing, grounded responses, and auditable traces.
              </p>
            </div>
            <ConversationList
              conversations={conversations}
              currentConversationId={conversationId}
              onNewChat={startNewChat}
              onSelect={(id) => void loadConversation(id)}
            />
          </div>
        </div>

        {/* Suggestion chips */}
        <SuggestionChips suggestions={suggestions} disabled={isSending} onSelect={(message) => void sendMessage(message)} />

        {/* Messages */}
        <ScrollArea className="flex-1 px-4 py-3">
          <div className="space-y-4">
            {messages.length === 0 ? (
              <div className="rounded-[1.75rem] border border-dashed bg-white/80 p-6">
                <p className="text-sm font-medium">Start with a direct question.</p>
                <p className="mt-2 text-sm text-muted-foreground">
                  Try &quot;What is our risk score?&quot;, &quot;Show critical alerts&quot;, or &quot;Build a dashboard for alerts and risk this week&quot;.
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

        {/* Footer status */}
        <div className="border-t bg-slate-50/70 px-4 py-2 text-xs text-muted-foreground">
          {conversationId ? `Conversation ${conversationId.slice(0, 8)} active` : 'New conversation'}
          {' · '}
          {preferredEngine === 'auto'
            ? 'Router decides per message'
            : preferredEngine === 'llm'
              ? 'LLM override active'
              : 'Deterministic override active'}
        </div>

        {/* Input */}
        <MessageInput
          value={input}
          preferredEngine={preferredEngine}
          onChange={setInput}
          onPreferredEngineChange={setPreferredEngine}
          onSend={() => void sendMessage(input)}
          disabled={isSending}
        />
      </div>

      {/* Confirm dialog for dangerous actions (replaces window.confirm) */}
      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmAction(null);
        }}
        title="Confirm Action"
        description={confirmAction?.params.warning ?? 'Do you want to continue?'}
        confirmLabel="Proceed"
        variant="default"
        onConfirm={handleConfirmAction}
      />
    </>
  );
}
