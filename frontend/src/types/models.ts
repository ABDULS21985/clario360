export interface User {
  id: string;
  tenant_id: string;
  email: string;
  first_name: string;
  last_name: string;
  is_active: boolean;
  roles: string[];
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  is_active: boolean;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  permissions: string[];
  is_system: boolean;
}

export interface AuditEntry {
  id: string;
  tenant_id: string;
  user_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  description: string;
  ip_address: string;
  service_name: string;
  created_at: string;
}

export type SuiteName = "cyber" | "data" | "acta" | "lex" | "visus";

export interface NavItem {
  title: string;
  href: string;
  icon?: string;
  children?: NavItem[];
  permission?: string;
}
