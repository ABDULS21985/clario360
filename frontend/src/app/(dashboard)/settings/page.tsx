"use client";

import { PageHeader } from "@/components/common/page-header";
import { ConnectedAccounts } from "@/components/auth/connected-accounts";
import { ProfileForm } from "./_components/profile-form";
import { PasswordChangeForm } from "./_components/password-change-form";
import { MFASection } from "./_components/mfa-section";
import { SessionsSection } from "./_components/sessions-section";
import { ApiKeysSection } from "./_components/api-keys-section";

export default function SettingsPage() {
  return (
    <div className="w-full space-y-6">
      <PageHeader
        title="Account Settings"
        description="Manage your profile, security, and API access"
      />
      <ProfileForm />
      <PasswordChangeForm />
      <MFASection />
      <ConnectedAccounts />
      <SessionsSection />
      <ApiKeysSection />
    </div>
  );
}
