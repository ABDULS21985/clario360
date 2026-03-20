import type { Metadata } from 'next';
import { DefinitionList } from './components/definition-list';

export const metadata: Metadata = {
  title: 'Workflow Definitions',
};

export default function WorkflowDefinitionsPage() {
  return <DefinitionList />;
}
