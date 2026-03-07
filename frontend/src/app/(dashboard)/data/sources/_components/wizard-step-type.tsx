'use client';

import { Activity, Cloud, Database, FileSpreadsheet, Globe } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { SourceTypeValue } from '@/lib/data-suite/forms';

const TYPES: Array<{
  value: SourceTypeValue | 'stream';
  title: string;
  description: string;
  icon: typeof Database;
  disabled?: boolean;
}> = [
  { value: 'postgresql', title: 'PostgreSQL', description: 'Connect to a PostgreSQL database', icon: Database },
  { value: 'mysql', title: 'MySQL', description: 'Connect to a MySQL database', icon: Database },
  { value: 'api', title: 'REST API', description: 'Connect to a REST API endpoint', icon: Globe },
  { value: 'csv', title: 'CSV / File', description: 'Import data from CSV or TSV files', icon: FileSpreadsheet },
  { value: 's3', title: 'S3 / MinIO', description: 'Connect to S3-compatible object storage', icon: Cloud },
  { value: 'stream', title: 'Streaming', description: 'Connect to a real-time data stream', icon: Activity, disabled: true },
];

interface WizardStepTypeProps {
  value?: SourceTypeValue;
  onSelect: (value: SourceTypeValue) => void;
}

export function WizardStepType({ value, onSelect }: WizardStepTypeProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {TYPES.map((type) => {
        const Icon = type.icon;
        const selected = value === type.value;

        return (
          <Card
            key={type.value}
            className={selected ? 'border-primary shadow-sm' : type.disabled ? 'opacity-70' : ''}
          >
            <CardHeader className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="rounded-full bg-primary/10 p-3 text-primary">
                  <Icon className="h-6 w-6" />
                </div>
                {type.disabled ? <Badge variant="outline">Coming soon</Badge> : null}
              </div>
              <div>
                <CardTitle className="text-lg">{type.title}</CardTitle>
                <CardDescription>{type.description}</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <Button
                type="button"
                className="w-full"
                variant={selected ? 'default' : 'outline'}
                disabled={type.disabled}
                onClick={() => {
                  if (type.value !== 'stream') {
                    onSelect(type.value);
                  }
                }}
              >
                Select →
              </Button>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
