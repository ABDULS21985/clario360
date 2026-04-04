'use client';

import { ChangeEvent } from 'react';
import { AlertCircle, Database, History, Upload } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { formatNumber } from '@/lib/format';
import type { AIValidationDatasetType, AIValidationPreview } from '@/types/ai-governance';

interface DatasetSelectorProps {
  datasetType: AIValidationDatasetType;
  timeRange: string;
  customText: string;
  customParseError: string | null;
  preview: AIValidationPreview | null;
  previewError: string | null;
  previewLoading: boolean;
  onDatasetTypeChange: (value: AIValidationDatasetType) => void;
  onTimeRangeChange: (value: string) => void;
  onCustomTextChange: (value: string) => void;
  onCustomFileLoad: (event: ChangeEvent<HTMLInputElement>) => void;
}

export function DatasetSelector({
  datasetType,
  timeRange,
  customText,
  customParseError,
  preview,
  previewError,
  previewLoading,
  onDatasetTypeChange,
  onTimeRangeChange,
  onCustomTextChange,
  onCustomFileLoad,
}: DatasetSelectorProps) {
  return (
    <Card className="border-border/70">
      <CardHeader>
        <CardTitle>Dataset Selection</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1.1fr_0.9fr]">
          <div className="space-y-2">
            <Label>Source</Label>
            <Select value={datasetType} onValueChange={(value) => onDatasetTypeChange(value as AIValidationDatasetType)}>
              <SelectTrigger>
                <SelectValue placeholder="Select a dataset" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="historical">
                  <div className="flex items-center gap-2">
                    <History className="h-4 w-4" />
                    Historical alerts with feedback
                  </div>
                </SelectItem>
                <SelectItem value="custom">
                  <div className="flex items-center gap-2">
                    <Upload className="h-4 w-4" />
                    Custom upload
                  </div>
                </SelectItem>
                <SelectItem value="live_replay" disabled>
                  <div className="flex items-center gap-2">
                    <Database className="h-4 w-4" />
                    Live replay (Unavailable)
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          {datasetType === 'historical' ? (
            <div className="space-y-2">
              <Label>Time Range</Label>
              <Select value={timeRange} onValueChange={onTimeRangeChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a time range" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="30d">Last 30 days</SelectItem>
                  <SelectItem value="90d">Last 90 days</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ) : null}
        </div>

        {datasetType === 'custom' ? (
          <div className="space-y-4 rounded-2xl border border-dashed border-border/80 bg-slate-50/70 p-4">
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-[0.45fr_0.55fr]">
              <div className="space-y-2">
                <Label htmlFor="validation-file">Upload JSON</Label>
                <Input id="validation-file" type="file" accept=".json,application/json" onChange={onCustomFileLoad} />
                <p className="text-sm text-muted-foreground">
                  Expected format: <code>[{'{'}&quot;input_hash&quot;: &quot;...&quot;, &quot;expected_label&quot;: &quot;threat&quot;{'}'}]</code>
                </p>
              </div>
              <div className="space-y-2">
                <Label>Paste JSON</Label>
                <Textarea
                  value={customText}
                  onChange={(event) => onCustomTextChange(event.target.value)}
                  rows={8}
                  className="font-mono text-xs"
                  placeholder='[{"input_hash":"...","expected_label":"threat"}]'
                />
              </div>
            </div>
            {customParseError ? (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>Custom dataset is invalid</AlertTitle>
                <AlertDescription>{customParseError}</AlertDescription>
              </Alert>
            ) : null}
          </div>
        ) : null}

        {preview ? (
          <div className="rounded-2xl border border-border/70 bg-white/70 p-4">
            <div className="text-sm text-slate-600">
              {formatNumber(preview.dataset_size)} samples ({formatNumber(preview.positive_count)} positive, {formatNumber(preview.negative_count)} negative)
            </div>
            {preview.dataset_size < 50 ? (
              <div className="mt-2 text-sm font-medium text-red-700">
                Validation is disabled until at least 50 labeled samples are available.
              </div>
            ) : null}
            {preview.warnings.map((warning) => (
              <div key={warning} className="mt-2 text-sm text-amber-700">
                {warning}
              </div>
            ))}
          </div>
        ) : null}

        {previewLoading ? <div className="text-sm text-muted-foreground">Checking labeled sample availability…</div> : null}
        {previewError ? (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Dataset preview failed</AlertTitle>
            <AlertDescription>{previewError}</AlertDescription>
          </Alert>
        ) : null}
      </CardContent>
    </Card>
  );
}
