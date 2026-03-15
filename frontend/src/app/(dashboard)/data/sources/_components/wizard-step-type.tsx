'use client';

import { useMemo, useState } from 'react';
import {
  BarChart3,
  Cloud,
  Database,
  FileSpreadsheet,
  Flame,
  GitBranch,
  GitCommit,
  Globe,
  HardDrive,
  Warehouse,
  Zap,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { SourceTypeValue } from '@/lib/data-suite/forms';

type SourceCategory = 'all' | 'database' | 'hadoop' | 'orchestration' | 'file_api';

const CATEGORIES: Array<{ value: SourceCategory; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'database', label: 'Databases' },
  { value: 'hadoop', label: 'Hadoop' },
  { value: 'orchestration', label: 'Orchestration' },
  { value: 'file_api', label: 'Files & API' },
];

const TYPES: Array<{
  value: Exclude<SourceTypeValue, 'stream'>;
  title: string;
  description: string;
  icon: typeof Database;
  category: Exclude<SourceCategory, 'all'>;
}> = [
  { value: 'postgresql', title: 'PostgreSQL', description: 'Relational operational database', icon: Database, category: 'database' },
  { value: 'mysql', title: 'MySQL', description: 'Relational operational database', icon: Database, category: 'database' },
  { value: 'clickhouse', title: 'ClickHouse', description: 'High-performance columnar analytics', icon: BarChart3, category: 'database' },
  { value: 'dolt', title: 'Dolt', description: 'Versioned SQL database with commit history', icon: GitCommit, category: 'database' },
  { value: 'impala', title: 'Apache Impala', description: 'Interactive SQL analytics for Cloudera', icon: Zap, category: 'hadoop' },
  { value: 'hive', title: 'Apache Hive', description: 'HiveServer2 warehouse over Hadoop storage', icon: Warehouse, category: 'hadoop' },
  { value: 'hdfs', title: 'HDFS', description: 'Direct Hadoop Distributed File System access', icon: HardDrive, category: 'hadoop' },
  { value: 'spark', title: 'Apache Spark', description: 'Distributed compute with SQL and job telemetry', icon: Flame, category: 'hadoop' },
  { value: 'dagster', title: 'Dagster', description: 'Pipeline orchestration and asset lineage', icon: GitBranch, category: 'orchestration' },
  { value: 'api', title: 'REST API', description: 'HTTP API endpoint integration', icon: Globe, category: 'file_api' },
  { value: 'csv', title: 'CSV / File', description: 'Delimited files in object storage', icon: FileSpreadsheet, category: 'file_api' },
  { value: 's3', title: 'S3 / MinIO', description: 'Object storage buckets and prefixes', icon: Cloud, category: 'file_api' },
];

interface WizardStepTypeProps {
  value?: SourceTypeValue;
  onSelect: (value: SourceTypeValue) => void;
}

export function WizardStepType({ value, onSelect }: WizardStepTypeProps) {
  const [category, setCategory] = useState<SourceCategory>('all');

  const visibleTypes = useMemo(
    () => TYPES.filter((type) => category === 'all' || type.category === category),
    [category],
  );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {CATEGORIES.map((item) => (
          <Button
            key={item.value}
            type="button"
            variant={category === item.value ? 'default' : 'outline'}
            size="sm"
            onClick={() => setCategory(item.value)}
          >
            {item.label}
          </Button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {visibleTypes.map((type) => {
          const Icon = type.icon;
          const selected = value === type.value;

          return (
            <Card key={type.value} className={selected ? 'border-primary shadow-sm' : ''}>
              <CardHeader className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="rounded-full bg-primary/10 p-3 text-primary">
                    <Icon className="h-6 w-6" />
                  </div>
                  <Badge variant="outline" className="capitalize">
                    {type.category === 'file_api' ? 'files & api' : type.category}
                  </Badge>
                </div>
                <div>
                  <CardTitle className="text-lg">{type.title}</CardTitle>
                  <CardDescription>{type.description}</CardDescription>
                </div>
              </CardHeader>
              <CardContent>
                <Button
                  type="button"
                  className="w-full"
                  variant={selected ? 'default' : 'outline'}
                  onClick={() => onSelect(type.value)}
                >
                  Select
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
