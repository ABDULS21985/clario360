import type { Metadata } from 'next';
import { TemplateDetailClient } from './template-detail-client';

export const metadata: Metadata = {
  title: 'Template Detail',
};

export default function TemplateDetailPage() {
  return <TemplateDetailClient />;
}
