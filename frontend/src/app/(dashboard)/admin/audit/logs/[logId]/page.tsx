"use client";

import { use } from "react";
import { LogDetail } from "./_components/log-detail";

interface AuditLogDetailPageProps {
  params: Promise<{ logId: string }>;
}

export default function AuditLogDetailPage({
  params,
}: AuditLogDetailPageProps) {
  const { logId } = use(params);

  return <LogDetail logId={logId} />;
}
