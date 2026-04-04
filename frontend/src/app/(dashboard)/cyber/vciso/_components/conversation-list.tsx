'use client';

import { History, MessageSquarePlus } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { cn, timeAgo } from '@/lib/utils';
import type { VCISOConversationListItem } from '@/types/cyber';

interface ConversationListProps {
  conversations: VCISOConversationListItem[];
  currentConversationId: string | null;
  onNewChat: () => void;
  onSelect: (conversationId: string) => void;
}

export function ConversationList({
  conversations,
  currentConversationId,
  onNewChat,
  onSelect,
}: ConversationListProps) {
  return (
    <div className="flex items-center gap-2">
      <Button type="button" variant="outline" size="sm" onClick={onNewChat}>
        <MessageSquarePlus className="mr-1.5 h-4 w-4" />
        New Chat
      </Button>
      <Sheet>
        <SheetTrigger asChild>
          <Button type="button" variant="outline" size="sm">
            <History className="mr-1.5 h-4 w-4" />
            History
          </Button>
        </SheetTrigger>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Conversation History</SheetTitle>
            <SheetDescription>Load a previous vCISO conversation.</SheetDescription>
          </SheetHeader>
          <ScrollArea className="mt-6 h-[calc(100vh-8rem)] pr-4">
            <div className="space-y-2">
              {conversations.length === 0 ? (
                <p className="rounded-xl border border-dashed p-4 text-sm text-muted-foreground">
                  No saved conversations yet.
                </p>
              ) : (
                conversations.map((conversation) => (
                  <button
                    key={conversation.id}
                    type="button"
                    onClick={() => onSelect(conversation.id)}
                    className={cn(
                      'w-full rounded-2xl border bg-white p-4 text-left transition-colors hover:border-primary/40 hover:bg-primary/5',
                      currentConversationId === conversation.id && 'border-primary bg-primary/5',
                    )}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold">{conversation.title}</p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {conversation.message_count} messages
                        </p>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {timeAgo(conversation.last_message_at ?? conversation.created_at)}
                      </span>
                    </div>
                  </button>
                ))
              )}
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>
    </div>
  );
}
