'use client';

import { SendHorizontal } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';

interface MessageInputProps {
  value: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  onSend: () => void;
}

export function MessageInput({ value, disabled = false, onChange, onSend }: MessageInputProps) {
  return (
    <div className="border-t bg-background/95 p-3">
      <div className="rounded-2xl border bg-white p-2 shadow-sm">
        <Textarea
          value={value}
          onChange={(event) => onChange(event.target.value.slice(0, 2000))}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault();
              onSend();
            }
          }}
          disabled={disabled}
          placeholder="Ask the vCISO..."
          className="min-h-[88px] resize-none border-0 bg-transparent px-2 py-2 shadow-none focus-visible:ring-0"
        />
        <div className="flex items-center justify-between px-2 pb-1">
          <span className="text-xs text-muted-foreground">{value.length}/2000</span>
          <Button onClick={onSend} disabled={disabled || value.trim().length === 0} size="sm" className="rounded-xl">
            <SendHorizontal className="mr-1.5 h-4 w-4" />
            Send
          </Button>
        </div>
      </div>
    </div>
  );
}
