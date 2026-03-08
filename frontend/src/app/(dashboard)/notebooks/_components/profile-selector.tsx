'use client';

import { Bot, Brain, ShieldAlert, Sparkles } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { NotebookProfile } from '@/lib/notebooks';

const iconMap = {
  'soc-analyst': ShieldAlert,
  'data-scientist': Brain,
  'spark-connected': Sparkles,
  admin: Bot,
} as const;

interface ProfileSelectorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profiles: NotebookProfile[];
  busy: boolean;
  onSelect: (profile: NotebookProfile) => void;
}

export function ProfileSelector({ open, onOpenChange, profiles, busy, onSelect }: ProfileSelectorProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Launch Notebook Workspace</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 md:grid-cols-2">
          {profiles.map((profile) => {
            const Icon = iconMap[profile.slug as keyof typeof iconMap] ?? Bot;
            return (
              <Card key={profile.slug} className="border-border/70">
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <div className="rounded-2xl bg-primary/10 p-3 text-primary">
                      <Icon className="h-5 w-5" />
                    </div>
                    <div>
                      <CardTitle className="text-base">{profile.display_name}</CardTitle>
                      <CardDescription>{profile.description}</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-3 gap-2 text-xs text-muted-foreground">
                    <div>CPU: {profile.cpu}</div>
                    <div>Memory: {profile.memory}</div>
                    <div>Storage: {profile.storage}</div>
                  </div>
                  <Button className="w-full" disabled={busy} onClick={() => onSelect(profile)}>
                    Launch {profile.display_name}
                  </Button>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
