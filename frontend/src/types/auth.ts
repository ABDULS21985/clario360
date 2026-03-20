import type { User, Tenant } from './models';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
  user: User;
  mfa_required?: undefined;
}

export interface MFARequiredResponse {
  mfa_required: true;
  mfa_token: string;
}

export type LoginApiResponse = LoginResponse | MFARequiredResponse;

export function isMFARequired(resp: LoginApiResponse): resp is MFARequiredResponse {
  return 'mfa_required' in resp && resp.mfa_required === true;
}

export interface VerifyMFARequest {
  mfa_token: string;
  code: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  tenant_name?: string;
  invite_code?: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_at?: string;
  token_type?: string;
  user?: User;
}

export interface ForgotPasswordRequest {
  email: string;
  tenant_id?: string;
}

export interface ResetPasswordRequest {
  token: string;
  new_password: string;
}

export interface EnableMFAResponse {
  otp_url: string;
  secret: string;
  recovery_codes: string[];
}

export interface TokenPayload {
  sub: string;
  email: string;
  tenant_id: string;
  roles: string[];
  permissions: string[];
  exp: number;
  iat: number;
  jti: string;
}

export interface SessionInfo {
  user: User;
  tenant: Tenant;
  expires_at: string;
}
