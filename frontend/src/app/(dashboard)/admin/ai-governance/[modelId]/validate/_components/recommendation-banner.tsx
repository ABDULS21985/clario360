'use client';

import { AlertTriangle, CheckCircle2, CircleX } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import type { AIValidationResult } from '@/types/ai-governance';

interface RecommendationBannerProps {
  result: AIValidationResult;
}

export function RecommendationBanner({ result }: RecommendationBannerProps) {
  switch (result.recommendation) {
    case 'promote':
      return (
        <Alert variant="success">
          <CheckCircle2 className="h-4 w-4" />
          <AlertTitle>Recommended for promotion</AlertTitle>
          <AlertDescription>{result.recommendation_reason}</AlertDescription>
        </Alert>
      );
    case 'reject':
      return (
        <Alert variant="destructive">
          <CircleX className="h-4 w-4" />
          <AlertTitle>Reject</AlertTitle>
          <AlertDescription>{result.recommendation_reason}</AlertDescription>
        </Alert>
      );
    default:
      return (
        <Alert variant="warning">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Keep testing</AlertTitle>
          <AlertDescription>{result.recommendation_reason}</AlertDescription>
        </Alert>
      );
  }
}
