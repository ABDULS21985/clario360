'use client';

import Link from 'next/link';
import { Calendar, Shield, Users } from 'lucide-react';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { committeeStatusConfig } from '@/lib/status-configs';
import { StatusBadge } from '@/components/shared/status-badge';
import type { ActaCommittee } from '@/types/suites';

interface CommitteeCardProps {
  committee: ActaCommittee;
}

export function CommitteeCard({ committee }: CommitteeCardProps) {
  const memberCount = committee.members?.filter((member) => member.active).length ?? 0;

  return (
    <Card className="h-full border-border/70 transition hover:border-primary/40 hover:shadow-sm">
      <CardHeader className="space-y-3">
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-lg">{committee.name}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground capitalize">
              {committee.type.replace(/_/g, ' ')}
            </p>
          </div>
          <StatusBadge status={committee.status} config={committeeStatusConfig} size="sm" />
        </div>
        <p className="line-clamp-3 text-sm text-muted-foreground">
          {committee.description}
        </p>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Users className="h-4 w-4" />
          <span>{memberCount} active members</span>
        </div>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Calendar className="h-4 w-4" />
          <span className="capitalize">{committee.meeting_frequency.replace(/_/g, ' ')}</span>
        </div>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Shield className="h-4 w-4" />
          <span>
            Quorum{' '}
            {committee.quorum_type === 'fixed_count'
              ? `${committee.quorum_fixed_count ?? 0} members`
              : `${committee.quorum_percentage}%`}
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          {committee.tags.slice(0, 3).map((tag) => (
            <Badge key={tag} variant="outline">
              {tag}
            </Badge>
          ))}
        </div>
      </CardContent>
      <CardFooter>
        <Button asChild className="w-full">
          <Link href={`/acta/committees/${committee.id}`}>Open Committee</Link>
        </Button>
      </CardFooter>
    </Card>
  );
}
