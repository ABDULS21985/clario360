'use client';

import { SendHorizontal } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import type { VCISOEnginePreference } from '@/types/cyber';

interface MessageInputProps {
  value: string;
  disabled?: boolean;
  preferredEngine: VCISOEnginePreference;
  onChange: (value: string) => void;
  onPreferredEngineChange: (value: VCISOEnginePreference) => void;
  onSend: () => void;
}

export function MessageInput({
  value,
  disabled = false,
  preferredEngine,
  onChange,
  onPreferredEngineChange,
  onSend,
}: MessageInputProps) {
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
        <div className="flex flex-col gap-3 px-2 pb-1 sm:flex-row sm:items-end sm:justify-between">
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">{value.length}/2000</span>
            <div className="w-full sm:w-[220px]">
              <Select value={preferredEngine} onValueChange={(value) => onPreferredEngineChange(value as VCISOEnginePreference)}>
                <SelectTrigger className="h-8 rounded-full border-dashed px-3 text-xs">
                  <SelectValue placeholder="Engine routing" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">Auto routing</SelectItem>
                  <SelectItem value="llm">Force LLM</SelectItem>
                  <SelectItem value="rule_based">Force deterministic</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <Button onClick={onSend} disabled={disabled || value.trim().length === 0} size="sm" className="rounded-xl">
            <SendHorizontal className="mr-1.5 h-4 w-4" />
            Send
          </Button>
        </div>
      </div>
    </div>
  );
}
