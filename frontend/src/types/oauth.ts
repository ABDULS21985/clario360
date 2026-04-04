export type OAuthProviderType = 'google' | 'github' | 'microsoft' | 'saml';

export interface OAuthProvider {
  provider: OAuthProviderType;
  enabled: boolean;
  display_name: string;
  icon_url: string;
  supports_pkce: boolean;
}

export interface OAuthConnection {
  id: string;
  provider: string;
  provider_user_id: string;
  provider_email: string;
  linked_at: string;
  last_login_at: string | null;
}
