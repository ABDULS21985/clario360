import type { Metadata } from 'next';
import { DefinitionsBrowserClient } from './definitions-browser-client';

export const metadata: Metadata = {
  title: 'Browse Workflows',
};

export default function WorkflowDefinitionsBrowserPage() {
  return <DefinitionsBrowserClient />;
}
