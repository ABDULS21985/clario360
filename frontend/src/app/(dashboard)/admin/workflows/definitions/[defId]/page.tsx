import type { Metadata } from 'next';
import { DefinitionDetailClient } from './components/definition-detail';

export const metadata: Metadata = {
  title: 'Workflow Definition',
};

export default function DefinitionDetailPage() {
  return <DefinitionDetailClient />;
}
