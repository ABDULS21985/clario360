"use client";

import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { useOAuthProviders, getOAuthAuthorizeUrl } from "@/hooks/use-oauth";
import { cn } from "@/lib/utils";

interface OAuthProvidersProps {
  className?: string;
}

const PROVIDER_ICONS: Record<string, React.FC<{ className?: string }>> = {
  google: ({ className }) => (
    <svg className={className} viewBox="0 0 24 24" aria-hidden>
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  ),
  github: ({ className }) => (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z"/>
    </svg>
  ),
  microsoft: ({ className }) => (
    <svg className={className} viewBox="0 0 24 24" aria-hidden>
      <path fill="#F25022" d="M1 1h10v10H1z"/>
      <path fill="#00A4EF" d="M1 13h10v10H1z"/>
      <path fill="#7FBA00" d="M13 1h10v10H13z"/>
      <path fill="#FFB900" d="M13 13h10v10H13z"/>
    </svg>
  ),
  saml: ({ className }) => (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M12 1a5 5 0 0 1 5 5v2a5 5 0 0 1-10 0V6a5 5 0 0 1 5-5zM7 6v2a5 5 0 0 0 10 0V6a5 5 0 0 0-10 0zm-3 8a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v5a3 3 0 0 1-3 3H7a3 3 0 0 1-3-3v-5zm3 0v5a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1v-5H7z"/>
    </svg>
  ),
};

export function OAuthProviders({ className }: OAuthProvidersProps) {
  const { data: providers, isLoading } = useOAuthProviders();

  const enabledProviders = providers?.filter((p) => p.enabled) ?? [];

  if (isLoading) {
    return (
      <div className={cn("flex justify-center py-2", className)}>
        <Spinner size="sm" />
      </div>
    );
  }

  if (enabledProviders.length === 0) {
    return null;
  }

  const handleOAuthLogin = (provider: string) => {
    const state = btoa(JSON.stringify({ provider, redirect_to: "/dashboard" }));
    const url = `${getOAuthAuthorizeUrl(provider)}?state=${state}`;
    window.location.href = url;
  };

  return (
    <div className={cn("space-y-3", className)}>
      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <span className="w-full border-t" />
        </div>
        <div className="relative flex justify-center text-xs uppercase">
          <span className="bg-background px-2 text-muted-foreground">Or continue with</span>
        </div>
      </div>

      <div className="grid gap-2">
        {enabledProviders.map((provider) => {
          const Icon = PROVIDER_ICONS[provider.provider];
          return (
            <Button
              key={provider.provider}
              variant="outline"
              type="button"
              className="w-full"
              onClick={() => handleOAuthLogin(provider.provider)}
            >
              {Icon && <Icon className="mr-2 h-4 w-4" />}
              {provider.display_name}
            </Button>
          );
        })}
      </div>
    </div>
  );
}
