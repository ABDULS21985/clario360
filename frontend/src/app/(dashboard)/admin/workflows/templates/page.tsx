import type { Metadata } from 'next';
import { TemplateGallery } from './components/template-gallery';

export const metadata: Metadata = {
  title: 'Workflow Templates',
};

export default function TemplatesPage() {
  return <TemplateGallery />;
}
