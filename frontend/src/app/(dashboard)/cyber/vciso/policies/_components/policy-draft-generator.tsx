'use client';

import { useState } from 'react';
import { Sparkles, Save, Loader2, FileText } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { parseApiError } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { PolicyDomain } from '@/types/cyber';

const POLICY_DOMAINS: { label: string; value: PolicyDomain }[] = [
  { label: 'Access Control', value: 'access_control' },
  { label: 'Incident Response', value: 'incident_response' },
  { label: 'Data Protection', value: 'data_protection' },
  { label: 'Acceptable Use', value: 'acceptable_use' },
  { label: 'Business Continuity', value: 'business_continuity' },
  { label: 'Risk Management', value: 'risk_management' },
  { label: 'Vendor Management', value: 'vendor_management' },
  { label: 'Change Management', value: 'change_management' },
  { label: 'Security Awareness', value: 'security_awareness' },
  { label: 'Network Security', value: 'network_security' },
  { label: 'Encryption', value: 'encryption' },
  { label: 'Physical Security', value: 'physical_security' },
  { label: 'Other', value: 'other' },
];

interface PolicyDraftGeneratorProps {
  onSaveAsDraft: (content: string, domain: PolicyDomain) => void;
}

export function PolicyDraftGenerator({ onSaveAsDraft }: PolicyDraftGeneratorProps) {
  const [domain, setDomain] = useState<PolicyDomain | ''>('');
  const [context, setContext] = useState('');
  const [generatedContent, setGeneratedContent] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);

  const handleGenerate = async () => {
    if (!domain) {
      toast.error('Please select a policy domain');
      return;
    }

    setIsGenerating(true);
    setGeneratedContent('');

    try {
      const result = await apiPost<{ content: string }>(
        API_ENDPOINTS.CYBER_VCISO_POLICY_GENERATE,
        {
          domain,
          context: context.trim() || undefined,
        },
      );
      setGeneratedContent(result.content);
      toast.success('Policy draft generated successfully');
    } catch (err) {
      toast.error(parseApiError(err));
    } finally {
      setIsGenerating(false);
    }
  };

  const handleSaveAsDraft = () => {
    if (!generatedContent) return;
    if (!domain) return;
    onSaveAsDraft(generatedContent, domain as PolicyDomain);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Sparkles className="h-5 w-5 text-primary" />
            AI Policy Draft Generator
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="draft-domain">Policy Domain</Label>
            <Select
              value={domain}
              onValueChange={(v) => setDomain(v as PolicyDomain)}
              disabled={isGenerating}
            >
              <SelectTrigger id="draft-domain">
                <SelectValue placeholder="Select a domain to generate a policy for" />
              </SelectTrigger>
              <SelectContent>
                {POLICY_DOMAINS.map((d) => (
                  <SelectItem key={d.value} value={d.value}>
                    {d.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="draft-context">
              Additional Context{' '}
              <span className="text-muted-foreground font-normal">(optional)</span>
            </Label>
            <Textarea
              id="draft-context"
              placeholder="Provide any specific requirements, industry standards, or organizational context to guide the draft generation..."
              value={context}
              onChange={(e) => setContext(e.target.value)}
              disabled={isGenerating}
              className="min-h-[120px]"
            />
          </div>

          <Button
            onClick={handleGenerate}
            disabled={isGenerating || !domain}
            className="w-full sm:w-auto"
          >
            {isGenerating ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Generate Draft
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Generated Content */}
      {(generatedContent || isGenerating) && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="flex items-center gap-2 text-lg">
              <FileText className="h-5 w-5 text-muted-foreground" />
              Generated Draft
            </CardTitle>
            {generatedContent && (
              <Button size="sm" onClick={handleSaveAsDraft}>
                <Save className="mr-1.5 h-3.5 w-3.5" />
                Save as Draft
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {isGenerating ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <Loader2 className="h-8 w-8 animate-spin text-primary mb-4" />
                <p className="text-sm text-muted-foreground">
                  Generating policy draft using AI...
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  This may take a moment
                </p>
              </div>
            ) : (
              <div
                className={cn(
                  'rounded-lg border border-border bg-muted/30 p-6',
                  'prose prose-sm max-w-none whitespace-pre-wrap text-sm leading-relaxed text-foreground',
                )}
              >
                {generatedContent}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Empty state when nothing is generated yet */}
      {!generatedContent && !isGenerating && (
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center justify-center text-center">
              <div className="rounded-full bg-primary/10 p-4 mb-4">
                <Sparkles className="h-8 w-8 text-primary" />
              </div>
              <h3 className="text-base font-medium mb-1">No draft generated yet</h3>
              <p className="text-sm text-muted-foreground max-w-sm">
                Select a policy domain and optionally provide additional context, then
                click Generate Draft to create an AI-powered policy document.
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
