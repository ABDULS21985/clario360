"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { AlertCircle, ShieldCheck, Workflow, Zap } from "lucide-react";

import {
  AuthActionStrip,
  AuthInsightGrid,
  AuthLoadingState,
  AuthPageIntro,
  type AuthInsightItem,
} from "@/components/auth/auth-page-primitives";
import { Alert, AlertDescription } from "@/components/ui/alert";
import api from "@/lib/api";
import { setAccessToken } from "@/lib/auth";
import { ROUTES } from "@/lib/constants";

const CALLBACK_INSIGHTS: AuthInsightItem[] = [
  {
    icon: ShieldCheck,
    label: "Identity exchange",
    value: "Federated callback",
    detail: "The external provider response is exchanged into the local access context.",
  },
  {
    icon: Zap,
    label: "Session handoff",
    value: "Immediate",
    detail: "A successful callback applies the token and routes directly into the workspace.",
  },
  {
    icon: Workflow,
    label: "Failure path",
    value: "Contained",
    detail: "If the provider callback fails, the user is returned to a controlled login entry point.",
  },
];

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
        const stateData = JSON.parse(atob(state as string));
        const provider = stateData.provider ?? "unknown";

        const { data } = await api.get<{ access_token: string; redirect_to?: string }>(
          `/api/v1/auth/oauth/${provider}/callback`,
          { params: { code, state } },
        );

        if (!cancelled) {
          if (data.access_token) {
            setAccessToken(data.access_token);
          }
          router.push(stateData.redirect_to ?? ROUTES.DASHBOARD);
        }
      } catch {
        if (!cancelled) {
          setError("Authentication failed. Please try again.");
        }
      }
    }

    void handleCallback();

    return () => {
      cancelled = true;
    };
  }, [searchParams, router]);

  if (error) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Federated callback"
          badgeIcon={AlertCircle}
          title="Authentication callback failed"
          description="The provider returned an error or the callback could not be completed in this browser session."
          statusLabel="Callback state"
          statusValue="Manual recovery required"
        />

        <AuthInsightGrid items={CALLBACK_INSIGHTS} />

        <Alert
          variant="destructive"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>

        <AuthActionStrip
          description="Restart the sign-in flow from the primary login surface."
          href={ROUTES.LOGIN}
          cta="Back to login"
        />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <AuthPageIntro
        badge="Federated callback"
        badgeIcon={ShieldCheck}
        title="Completing secure sign-in"
        description="The provider response is being exchanged for an application session and your workspace context is about to load."
        statusLabel="Callback state"
        statusValue="Finalizing session"
      />

      <AuthInsightGrid items={CALLBACK_INSIGHTS} />

      <AuthLoadingState
        label="Completing sign-in"
        detail="We are validating the provider response, applying the access token, and redirecting to the workspace."
      />

      <div className="text-center">
        <Link
          href={ROUTES.LOGIN}
          className="text-sm font-medium text-slate-500 hover:text-slate-900 hover:underline"
        >
          Return to login instead
        </Link>
      </div>
    </div>
  );
}
