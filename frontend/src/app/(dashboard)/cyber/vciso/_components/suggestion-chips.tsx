'use client';

import { Button } from '@/components/ui/button';
import type { VCISOSuggestion } from '@/types/cyber';

interface SuggestionChipsProps {
  suggestions: VCISOSuggestion[];
  disabled?: boolean;
  onSelect: (message: string) => void;
}

export function SuggestionChips({ suggestions, disabled = false, onSelect }: SuggestionChipsProps) {
  if (suggestions.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-2 overflow-x-auto px-3 py-3">
      {suggestions.map((suggestion) => (
        <Button
          key={`${suggestion.text}-${suggestion.priority}`}
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          className="h-auto rounded-full bg-white px-3 py-2 text-left whitespace-nowrap"
          onClick={() => onSelect(suggestion.text)}
        >
          {suggestion.text}
        </Button>
      ))}
    </div>
  );
}
