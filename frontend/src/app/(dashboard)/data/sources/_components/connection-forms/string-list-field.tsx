'use client';

import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FormField } from '@/components/shared/forms/form-field';

interface StringListFieldProps {
  name: string;
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder: string;
  description?: string;
  required?: boolean;
}

export function StringListField({
  name,
  label,
  values,
  onChange,
  placeholder,
  description,
  required = false,
}: StringListFieldProps) {
  return (
    <FormField name={name} label={label} description={description} required={required}>
      <div className="space-y-2">
        {values.map((value, index) => (
          <div key={`${name}-${index}`} className="flex items-center gap-2">
            <Input
              value={value}
              onChange={(event) => onChange(values.map((item, itemIndex) => (itemIndex === index ? event.target.value : item)))}
              placeholder={placeholder}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              disabled={values.length === 1}
              onClick={() => onChange(values.filter((_, itemIndex) => itemIndex !== index))}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...values, ''])}>
          <Plus className="mr-1.5 h-4 w-4" />
          Add value
        </Button>
      </div>
    </FormField>
  );
}
