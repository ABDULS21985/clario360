export interface Invitation {
  id: string;
  tenant_id: string;
  email: string;
  role_id: string;
  role_name: string;
  status: InvitationStatus;
  message: string | null;
  invited_by: string;
  inviter_name: string;
  expires_at: string;
  accepted_at: string | null;
  created_at: string;
}

export type InvitationStatus = 'pending' | 'accepted' | 'expired' | 'cancelled';

export interface CreateInvitationRequest {
  email: string;
  role_id: string;
  message?: string;
  expires_in_days?: number;
}

export interface InvitationStats {
  total_sent: number;
  pending: number;
  accepted: number;
  expired: number;
  acceptance_rate: number;
}
