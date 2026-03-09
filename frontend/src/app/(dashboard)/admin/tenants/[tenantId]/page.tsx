"use client";

import { use } from "react";
import { TenantDetailContent } from "./_components/tenant-detail";

interface TenantDetailPageProps {
  params: Promise<{ tenantId: string }>;
}

export default function TenantDetailPage({ params }: TenantDetailPageProps) {
  const { tenantId } = use(params);
  return <TenantDetailContent tenantId={tenantId} />;
}
