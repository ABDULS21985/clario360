export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  scopes: ApiKeyScope[];
  status: ApiKeyStatus;
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
  created_by: string | null;
}

export type ApiKeyStatus = 'active' | 'revoked' | 'expired';

export type ApiKeyScope =
  | 'read:users' | 'write:users'
  | 'read:cyber' | 'write:cyber'
  | 'read:data' | 'write:data'
  | 'read:acta' | 'write:acta'
  | 'read:lex' | 'write:lex'
  | 'read:visus' | 'write:visus'
  | 'admin:tenants' | 'admin:audit';

export interface CreateApiKeyRequest {
  name: string;
  scopes: ApiKeyScope[];
  expires_at: string | null;
}

export interface CreateApiKeyResponse {
  key: ApiKey;
  secret: string;
}

export const API_KEY_SCOPE_GROUPS: { label: string; scopes: ApiKeyScope[] }[] = [
  { label: 'Users', scopes: ['read:users', 'write:users'] },
  { label: 'Cybersecurity', scopes: ['read:cyber', 'write:cyber'] },
  { label: 'Data Intelligence', scopes: ['read:data', 'write:data'] },
  { label: 'Acta (Governance)', scopes: ['read:acta', 'write:acta'] },
  { label: 'Lex (Legal)', scopes: ['read:lex', 'write:lex'] },
  { label: 'Visus (Executive)', scopes: ['read:visus', 'write:visus'] },
  { label: 'Admin', scopes: ['admin:tenants', 'admin:audit'] },
];
