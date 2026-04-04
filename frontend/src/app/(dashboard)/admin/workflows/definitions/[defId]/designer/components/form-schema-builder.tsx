'use client';

import { useCallback } from 'react';
import { Plus, Trash2, GripVertical } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { FormField, FormFieldType } from '@/types/models';

const FIELD_TYPES: { value: FormFieldType; label: string }[] = [
  { value: 'text', label: 'Text' },
  { value: 'textarea', label: 'Text Area' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Checkbox' },
  { value: 'date', label: 'Date' },
  { value: 'select', label: 'Select' },
];

interface FormSchemaBuilderProps {
  fields: FormField[];
  onChange: (fields: FormField[]) => void;
  readOnly?: boolean;
}

export function FormSchemaBuilder({ fields, onChange, readOnly }: FormSchemaBuilderProps) {
  const handleAdd = useCallback(() => {
    const name = `field_${fields.length + 1}`;
    onChange([
      ...fields,
      {
        name,
        type: 'text',
        label: '',
        required: false,
        placeholder: '',
      },
    ]);
  }, [fields, onChange]);

  const handleRemove = useCallback(
    (index: number) => {
      onChange(fields.filter((_, i) => i !== index));
    },
    [fields, onChange],
  );

  const handleUpdate = useCallback(
    (index: number, updates: Partial<FormField>) => {
      onChange(
        fields.map((f, i) => (i === index ? { ...f, ...updates } : f)),
      );
    },
    [fields, onChange],
  );

  const handleMove = useCallback(
    (index: number, direction: -1 | 1) => {
      const target = index + direction;
      if (target < 0 || target >= fields.length) return;
      const updated = [...fields];
      [updated[index], updated[target]] = [updated[target], updated[index]];
      onChange(updated);
    },
    [fields, onChange],
  );

  return (
    <div className="space-y-2">
      <Label className="text-xs">Form Fields</Label>

      {fields.length === 0 && (
        <p className="text-xs text-muted-foreground">
          No form fields. The task will have no input form.
        </p>
      )}

      {fields.map((field, i) => (
        <div key={i} className="space-y-1.5 rounded-md border bg-muted/30 p-2">
          <div className="flex items-center gap-1.5">
            {/* Reorder */}
            {!readOnly && (
              <div className="flex flex-col -my-1">
                <button
                  type="button"
                  className="text-muted-foreground hover:text-foreground disabled:opacity-30"
                  disabled={i === 0}
                  onClick={() => handleMove(i, -1)}
                  aria-label="Move up"
                >
                  <GripVertical className="h-3 w-3 rotate-90" />
                </button>
                <button
                  type="button"
                  className="text-muted-foreground hover:text-foreground disabled:opacity-30"
                  disabled={i === fields.length - 1}
                  onClick={() => handleMove(i, 1)}
                  aria-label="Move down"
                >
                  <GripVertical className="h-3 w-3 -rotate-90" />
                </button>
              </div>
            )}

            {/* Name */}
            <Input
              value={field.name}
              onChange={(e) => handleUpdate(i, { name: e.target.value })}
              placeholder="field_name"
              disabled={readOnly}
              className="h-7 text-xs flex-1 font-mono"
            />

            {/* Type */}
            <Select
              value={field.type}
              onValueChange={(v) =>
                handleUpdate(i, { type: v as FormFieldType })
              }
              disabled={readOnly}
            >
              <SelectTrigger className="h-7 w-24 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FIELD_TYPES.map((ft) => (
                  <SelectItem key={ft.value} value={ft.value}>
                    {ft.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {/* Required toggle */}
            <div className="flex items-center gap-1">
              <Switch
                checked={field.required}
                onCheckedChange={(v) => handleUpdate(i, { required: v })}
                disabled={readOnly}
                className="scale-75"
              />
              <span className="text-[10px] text-muted-foreground">Req</span>
            </div>

            {/* Remove */}
            {!readOnly && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                onClick={() => handleRemove(i)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
          </div>

          {/* Label */}
          <Input
            value={field.label}
            onChange={(e) => handleUpdate(i, { label: e.target.value })}
            placeholder="Display label"
            disabled={readOnly}
            className="h-7 text-xs"
          />

          {/* Placeholder */}
          <Input
            value={field.placeholder ?? ''}
            onChange={(e) => handleUpdate(i, { placeholder: e.target.value })}
            placeholder="Placeholder text"
            disabled={readOnly}
            className="h-7 text-xs"
          />

          {/* Options (for select type) */}
          {field.type === 'select' && (
            <Input
              value={(field.options ?? []).join(', ')}
              onChange={(e) =>
                handleUpdate(i, {
                  options: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
              placeholder="option1, option2, option3"
              disabled={readOnly}
              className="h-7 text-xs"
            />
          )}
        </div>
      ))}

      {!readOnly && (
        <Button
          variant="outline"
          size="sm"
          className="w-full h-7 text-xs"
          onClick={handleAdd}
        >
          <Plus className="mr-1 h-3 w-3" />
          Add Field
        </Button>
      )}
    </div>
  );
}
