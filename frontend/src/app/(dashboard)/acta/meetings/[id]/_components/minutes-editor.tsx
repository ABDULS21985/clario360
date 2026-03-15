'use client';

import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';

interface MinutesEditorProps {
  initialValue: string;
  onSave: (content: string) => void;
  onCancel: () => void;
  pending?: boolean;
}

export function MinutesEditor({
  initialValue,
  onSave,
  onCancel,
  pending = false,
}: MinutesEditorProps) {
  const [content, setContent] = useState(initialValue);

  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div className="space-y-3">
        <Textarea value={content} onChange={(event) => setContent(event.target.value)} rows={24} />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={() => onSave(content)} disabled={pending}>
            {pending ? 'Saving…' : 'Save minutes'}
          </Button>
        </div>
      </div>
      <div className="rounded-xl border bg-card p-4">
        <p className="mb-3 text-sm font-medium">Preview</p>
        <article className="prose prose-sm max-w-none dark:prose-invert">
          <ReactMarkdown>{content}</ReactMarkdown>
        </article>
      </div>
    </div>
  );
}
