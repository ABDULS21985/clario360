import type { Metadata } from 'next';
import { DesignerPageClient } from './designer-page-client';

export const metadata: Metadata = {
  title: 'Workflow Designer',
};

export default function DesignerPage() {
  return <DesignerPageClient />;
}
