'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { StatusBadge } from '@/components/shared/status-badge';
import { workflowDefinitionStatusConfig } from '@/lib/status-configs';
import { useWorkflowDefinitions } from '@/hooks/use-workflow-definitions';
import { titleCase } from '@/lib/format';

export function TaskWorkloadTable() {
  const { data, isLoading, isError } = useWorkflowDefinitions({
    per_page: 20,
    sort: 'instance_count',
    order: 'desc',
  });

  const definitions = data?.data ?? [];

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">
          Workflow Definitions by Usage
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <LoadingSkeleton variant="table-row" count={5} />
        ) : isError ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            Failed to load definitions.
          </p>
        ) : definitions.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            No definitions found.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Category</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Steps</TableHead>
                <TableHead className="text-right">Instances</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {definitions.map((def) => (
                <TableRow key={def.id}>
                  <TableCell className="font-medium">{def.name}</TableCell>
                  <TableCell>
                    {def.category ? (
                      <Badge variant="outline" className="text-xs">
                        {titleCase(def.category)}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={def.status}
                      config={workflowDefinitionStatusConfig}
                    />
                  </TableCell>
                  <TableCell className="text-right text-sm">
                    {def.step_count ?? 0}
                  </TableCell>
                  <TableCell className="text-right text-sm font-medium">
                    {def.instance_count ?? 0}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
