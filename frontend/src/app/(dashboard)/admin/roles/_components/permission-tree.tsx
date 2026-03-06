'use client';

import { useState, useMemo } from 'react';
import { ChevronRight } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { cn } from '@/lib/utils';

interface PermissionNode {
  label: string;
  value: string; // the wildcard or specific permission
  children?: PermissionNode[];
}

const PERMISSION_TREE: PermissionNode[] = [
  {
    label: 'Cybersecurity',
    value: 'cyber:*',
    children: [
      { label: 'Read', value: 'cyber:read' },
      { label: 'Write', value: 'cyber:write' },
      {
        label: 'Alerts',
        value: 'alerts:*',
        children: [
          { label: 'Read alerts', value: 'alerts:read' },
          { label: 'Manage alerts', value: 'alerts:write' },
        ],
      },
      {
        label: 'Remediation',
        value: 'remediation:*',
        children: [
          { label: 'View', value: 'remediation:read' },
          { label: 'Execute', value: 'remediation:execute' },
          { label: 'Approve', value: 'remediation:approve' },
        ],
      },
    ],
  },
  {
    label: 'Data Intelligence',
    value: 'data:*',
    children: [
      { label: 'Read', value: 'data:read' },
      { label: 'Write', value: 'data:write' },
      {
        label: 'Pipelines',
        value: 'pipelines:*',
        children: [
          { label: 'Read', value: 'pipelines:read' },
          { label: 'Write', value: 'pipelines:write' },
        ],
      },
      { label: 'Quality', value: 'quality:*' },
      { label: 'Lineage', value: 'lineage:*' },
    ],
  },
  {
    label: 'Governance — Acta',
    value: 'acta:*',
    children: [
      { label: 'Read', value: 'acta:read' },
      { label: 'Write', value: 'acta:write' },
    ],
  },
  {
    label: 'Legal — Lex',
    value: 'lex:*',
    children: [
      { label: 'Read', value: 'lex:read' },
      { label: 'Write', value: 'lex:write' },
    ],
  },
  {
    label: 'Executive — Visus',
    value: 'visus:*',
    children: [
      { label: 'Read', value: 'visus:read' },
      { label: 'Write', value: 'visus:write' },
    ],
  },
  {
    label: 'Reports',
    value: 'reports:*',
    children: [{ label: 'Read', value: 'reports:read' }],
  },
  {
    label: 'Platform Admin',
    value: 'platform:*',
    children: [
      { label: 'Tenant', value: 'tenant:*' },
      {
        label: 'Users',
        value: 'users:*',
        children: [
          { label: 'Read', value: 'users:read' },
          { label: 'Write', value: 'users:write' },
          { label: 'Delete', value: 'users:delete' },
        ],
      },
      {
        label: 'Roles',
        value: 'roles:*',
        children: [
          { label: 'Read', value: 'roles:read' },
          { label: 'Write', value: 'roles:write' },
          { label: 'Delete', value: 'roles:delete' },
          { label: 'Assign', value: 'roles:assign' },
        ],
      },
      { label: 'API Keys', value: 'apikeys:*' },
    ],
  },
  { label: 'Full Access', value: '*' },
];

function getAllLeafValues(node: PermissionNode): string[] {
  if (!node.children || node.children.length === 0) return [node.value];
  return node.children.flatMap(getAllLeafValues);
}

function getAllNodeValues(node: PermissionNode): string[] {
  const values = [node.value];
  if (node.children) {
    values.push(...node.children.flatMap(getAllNodeValues));
  }
  return values;
}

type CheckState = boolean | 'indeterminate';

function getNodeCheckState(node: PermissionNode, selected: Set<string>): CheckState {
  // If the node's own wildcard is selected, it's fully checked
  if (selected.has(node.value)) return true;
  if (!node.children || node.children.length === 0) {
    return selected.has(node.value);
  }
  const childStates = node.children.map((c) => getNodeCheckState(c, selected));
  const allChecked = childStates.every((s) => s === true);
  const noneChecked = childStates.every((s) => s === false);
  if (allChecked) return true;
  if (noneChecked) return false;
  return 'indeterminate';
}

function matchesSearch(node: PermissionNode, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  if (node.label.toLowerCase().includes(q) || node.value.toLowerCase().includes(q)) return true;
  return node.children?.some((c) => matchesSearch(c, query)) ?? false;
}

interface PermissionTreeNodeProps {
  node: PermissionNode;
  selected: Set<string>;
  onToggle: (node: PermissionNode, checked: boolean) => void;
  depth?: number;
  search: string;
}

function PermissionTreeNode({ node, selected, onToggle, depth = 0, search }: PermissionTreeNodeProps) {
  const [expanded, setExpanded] = useState(true);

  if (search && !matchesSearch(node, search)) return null;

  const hasChildren = node.children && node.children.length > 0;
  const checkState = getNodeCheckState(node, selected);
  const isFullAccess = node.value === '*';

  return (
    <div>
      <div
        className={cn(
          'flex items-center gap-2 rounded px-2 py-1.5 hover:bg-muted/50 cursor-pointer',
          depth > 0 && 'ml-4'
        )}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
      >
        {hasChildren && (
          <button
            type="button"
            onClick={() => setExpanded((e) => !e)}
            className="h-4 w-4 shrink-0 text-muted-foreground hover:text-foreground"
            aria-label={expanded ? 'Collapse' : 'Expand'}
          >
            <ChevronRight
              className={cn('h-4 w-4 transition-transform', expanded && 'rotate-90')}
            />
          </button>
        )}
        {!hasChildren && <span className="h-4 w-4 shrink-0" />}
        <Checkbox
          checked={checkState === 'indeterminate' ? 'indeterminate' : checkState}
          onCheckedChange={(checked) => onToggle(node, !!checked)}
          onClick={(e) => e.stopPropagation()}
          id={`perm-${node.value}`}
        />
        <label
          htmlFor={`perm-${node.value}`}
          className="flex-1 cursor-pointer text-sm select-none"
        >
          {node.label}
        </label>
        <span className="text-xs font-mono text-muted-foreground">{node.value}</span>
      </div>

      {hasChildren && expanded && (
        <div>
          {node.children!.map((child) => (
            <PermissionTreeNode
              key={child.value}
              node={child}
              selected={selected}
              onToggle={onToggle}
              depth={depth + 1}
              search={search}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface PermissionTreeProps {
  value: string[];
  onChange: (permissions: string[]) => void;
}

export function PermissionTree({ value, onChange }: PermissionTreeProps) {
  const [search, setSearch] = useState('');
  const selected = useMemo(() => new Set(value), [value]);

  const handleToggle = (node: PermissionNode, checked: boolean) => {
    const next = new Set(selected);

    if (node.value === '*') {
      if (checked) {
        PERMISSION_TREE.forEach((n) => {
          getAllNodeValues(n).forEach((v) => next.add(v));
        });
      } else {
        next.clear();
      }
      onChange(Array.from(next));
      return;
    }

    if (checked) {
      // Select this node's wildcard and remove individual children
      if (node.children) {
        // Remove all individual child values (they're covered by wildcard)
        getAllNodeValues(node).forEach((v) => next.delete(v));
        next.add(node.value);
      } else {
        next.add(node.value);
      }
    } else {
      // Deselect: remove this node and all its children
      getAllNodeValues(node).forEach((v) => next.delete(v));
    }

    // Remove parent wildcard if partially selected
    PERMISSION_TREE.forEach((root) => {
      getAllNodeValues(root).forEach((v) => {
        if (v.endsWith(':*') || v === '*') {
          // If parent wildcard is set but not all children selected, remove it
          const parentNode = findNode(PERMISSION_TREE, v);
          if (parentNode && next.has(v)) {
            const allChildren = getAllLeafValues(parentNode);
            const allSelected = allChildren.every((lv) => next.has(lv) || next.has(v));
            if (!allSelected) {
              // This wildcard is being partially overridden, keep it as is
            }
          }
        }
      });
    });

    onChange(Array.from(next));
  };

  const selectedCount = value.length;

  return (
    <div className="space-y-3">
      <Input
        placeholder="Search permissions..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="h-8"
      />

      {value.includes('*') && (
        <Alert>
          <AlertDescription className="text-xs">
            ⚠️ Full Access grants unrestricted access to the entire platform.
          </AlertDescription>
        </Alert>
      )}

      <div className="rounded-md border max-h-72 overflow-y-auto">
        {PERMISSION_TREE.map((node) => (
          <PermissionTreeNode
            key={node.value}
            node={node}
            selected={selected}
            onToggle={handleToggle}
            search={search}
          />
        ))}
      </div>

      <p className="text-xs text-muted-foreground">
        {selectedCount === 0
          ? 'No permissions selected'
          : `${selectedCount} permission${selectedCount !== 1 ? 's' : ''} selected`}
      </p>
    </div>
  );
}

function findNode(nodes: PermissionNode[], value: string): PermissionNode | null {
  for (const node of nodes) {
    if (node.value === value) return node;
    if (node.children) {
      const found = findNode(node.children, value);
      if (found) return found;
    }
  }
  return null;
}
