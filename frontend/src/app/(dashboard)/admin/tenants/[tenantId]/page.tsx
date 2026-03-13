"use client";


import { TenantDetailContent } from "./_components/tenant-detail";

interface TenantDetailPageProps {
  params: { tenantId: string };
}

export default function TenantDetailPage({ params }: TenantDetailPageProps) {
  const { tenantId } = params;
  return <TenantDetailContent tenantId={tenantId} />;
}
