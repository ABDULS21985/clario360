"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Spinner } from "@/components/ui/spinner";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import api from "@/lib/api";
import { setAccessToken } from "@/lib/auth";
import Link from "next/link";

export default function OAuthCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const code = searchParams?.get("code");
    const state = searchParams?.get("state");
    const errorParam = searchParams?.get("error");
    const errorDescription = searchParams?.get("error_description");

    if (errorParam) {
      setError(errorDescription ?? errorParam);
      return;
    }

    if (!code || !state) {
      setError("Missing authorization parameters. Please try again.");
      return;
    }

    let cancelled = false;

    async function handleCallback() {
      try {
        // The state parameter encodes the provider and redirect info
        const stateData = JSON.parse(atob(state!));
        const provider = stateData.provider ?? "unknown";

        const { data } = await api.get<{ access_token: string; redirect_to?: string }>(
          `/api/v1/auth/oauth/${provider}/callback`,
          { params: { code, state } },
        );

        if (!cancelled) {
          if (data.access_token) {
            setAccessToken(data.access_token);
          }
          router.push(stateData.redirect_to ?? "/dashboard");
        }
      } catch {
        if (!cancelled) {
          setError("Authentication failed. Please try again.");
        }
      }
    }

    handleCallback();

    return () => {
      cancelled = true;
    };
  }, [searchParams, router]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="w-full max-w-sm space-y-6 p-6">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
          <Button asChild className="w-full">
            <Link href="/login">Back to Login</Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <Spinner size="lg" />
        <p className="text-sm text-muted-foreground">Completing sign in...</p>
      </div>
    </div>
  );
}
