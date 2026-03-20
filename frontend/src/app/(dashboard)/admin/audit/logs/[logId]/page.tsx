"use client";


import { LogDetail } from "./_components/log-detail";

interface AuditLogDetailPageProps {
  params: { logId: string };
}

export default function AuditLogDetailPage({
  params,
}: AuditLogDetailPageProps) {
  const { logId } = params;

  return <LogDetail logId={logId} />;
}
