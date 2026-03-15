export interface Invitation {
  id: string;
  tenant_id: string;
  email: string;
  role_slug: string;
  role_name: string;
  status: InvitationStatus;
  message: string | null;
  invited_by: string;
  invited_by_name: string;
  expires_at: string;
  accepted_at: string | null;
  created_at: string;
}

export type InvitationStatus = 'pending' | 'accepted' | 'expired' | 'cancelled' | 'revoked';

export interface CreateInvitationRequest {
  invitations: Array<{
    email: string;
    role_slug: string;
    message?: string;
  }>;
}

export interface InvitationStats {
  total_sent: number;
  pending: number;
  accepted: number;
  expired: number;
  acceptance_rate: number;
}
