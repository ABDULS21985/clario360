'use client';

import { BookOpen, Brain, Shield, Sparkles } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { NotebookServer, NotebookTemplate } from '@/lib/notebooks';

const iconForTemplate = (tags: string[]) => {
  if (tags.includes('spark')) return Sparkles;
  if (tags.includes('ai')) return Brain;
  if (tags.includes('security')) return Shield;
  return BookOpen;
};

interface TemplateGalleryProps {
  templates: NotebookTemplate[];
  activeServer: NotebookServer | null;
  busyTemplateId: string | null;
  onOpenTemplate: (template: NotebookTemplate) => void;
}

export function TemplateGallery({ templates, activeServer, busyTemplateId, onOpenTemplate }: TemplateGalleryProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {templates.map((template) => {
        const Icon = iconForTemplate(template.tags);
        return (
          <Card key={template.id} className="border-border/70">
            <CardHeader>
              <div className="flex items-start justify-between gap-3">
                <div className="rounded-2xl bg-primary/10 p-3 text-primary">
                  <Icon className="h-5 w-5" />
                </div>
                <Badge variant="outline">{template.difficulty}</Badge>
              </div>
              <CardTitle className="text-base">{template.title}</CardTitle>
              <CardDescription>{template.description}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                {template.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
              <Button
                className="w-full"
                disabled={!activeServer || busyTemplateId === template.id}
                onClick={() => onOpenTemplate(template)}
              >
                {activeServer ? 'Open Template' : 'Launch a server first'}
              </Button>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
