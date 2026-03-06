'use client';

import { PageHeader } from '@/components/common/page-header';
import { ProfileForm } from './_components/profile-form';
import { PasswordChangeForm } from './_components/password-change-form';
import { MFASection } from './_components/mfa-section';
import { SessionsSection } from './_components/sessions-section';
import { ApiKeysSection } from './_components/api-keys-section';

export default function SettingsPage() {
  return (
    <div className="space-y-6 max-w-2xl">
      <PageHeader
        title="Profile Settings"
        description="Manage your account, security, and preferences"
      />

      <ProfileForm />
      <PasswordChangeForm />
      <MFASection />
      <SessionsSection />
      <ApiKeysSection />
    </div>
  );
}
