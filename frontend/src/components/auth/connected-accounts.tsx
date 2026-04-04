"use client";

import { useState } from "react";
import { Link2, Unlink, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { RelativeTime } from "@/components/shared/relative-time";
import {
  useOAuthProviders,
  useOAuthConnections,
  useUnlinkOAuth,
  getOAuthAuthorizeUrl,
} from "@/hooks/use-oauth";
import type { OAuthConnection } from "@/types/oauth";

const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  github: "GitHub",
  microsoft: "Microsoft",
  saml: "SAML SSO",
};

export function ConnectedAccounts() {
  const { data: providers, isLoading: providersLoading } = useOAuthProviders();
  const { data: connections, isLoading: connectionsLoading } = useOAuthConnections();
  const unlinkMutation = useUnlinkOAuth();
  const [unlinkProvider, setUnlinkProvider] = useState<OAuthConnection | null>(null);

  const isLoading = providersLoading || connectionsLoading;
  const enabledProviders = providers?.filter((p) => p.enabled) ?? [];
  const connectedProviders = new Set(connections?.map((c) => c.provider) ?? []);

  const handleConnect = (provider: string) => {
    const state = btoa(JSON.stringify({ provider, redirect_to: "/settings", action: "link" }));
    const url = `${getOAuthAuthorizeUrl(provider)}?state=${state}&action=link`;
    window.location.href = url;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Link2 className="h-5 w-5" />
          Connected Accounts
        </CardTitle>
        <CardDescription>
          Link external accounts for single sign-on
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {isLoading ? (
            Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-14 rounded" />
            ))
          ) : enabledProviders.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No external authentication providers are configured.
            </p>
          ) : (
            enabledProviders.map((provider) => {
              const connection = connections?.find((c) => c.provider === provider.provider);
              const isConnected = !!connection;

              return (
                <div
                  key={provider.provider}
                  className="flex items-center justify-between gap-4 rounded-lg border border-border p-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted">
                      <span className="text-xs font-bold uppercase">
                        {provider.provider.slice(0, 2)}
                      </span>
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium">
                          {PROVIDER_LABELS[provider.provider] ?? provider.display_name}
                        </p>
                        {isConnected && (
                          <Badge variant="outline" className="text-xs">
                            Connected
                          </Badge>
                        )}
                      </div>
                      {connection && (
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {connection.provider_email}
                          {connection.last_login_at && (
                            <>
                              {" · Last used "}
                              <RelativeTime date={connection.last_login_at} />
                            </>
                          )}
                        </p>
                      )}
                    </div>
                  </div>

                  {isConnected ? (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setUnlinkProvider(connection)}
                    >
                      <Unlink className="mr-1 h-3 w-3" />
                      Unlink
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleConnect(provider.provider)}
                    >
                      <ExternalLink className="mr-1 h-3 w-3" />
                      Connect
                    </Button>
                  )}
                </div>
              );
            })
          )}
        </div>
      </CardContent>

      {unlinkProvider && (
        <ConfirmDialog
          open={!!unlinkProvider}
          onOpenChange={(o) => !o && setUnlinkProvider(null)}
          title="Unlink Account"
          description={`Unlink your ${PROVIDER_LABELS[unlinkProvider.provider] ?? unlinkProvider.provider} account (${unlinkProvider.provider_email})? You will no longer be able to sign in with this provider.`}
          confirmLabel="Unlink"
          variant="destructive"
          loading={unlinkMutation.isPending}
          onConfirm={async () => {
            await unlinkMutation.mutateAsync(unlinkProvider.provider);
          }}
        />
      )}
    </Card>
  );
}
