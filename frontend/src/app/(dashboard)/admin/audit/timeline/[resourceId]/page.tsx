"use client";

import { use } from "react";
import { ResourceTimeline } from "./_components/resource-timeline";

interface ResourceTimelinePageProps {
  params: Promise<{ resourceId: string }>;
}

export default function ResourceTimelinePage({
  params,
}: ResourceTimelinePageProps) {
  const { resourceId } = use(params);

  return <ResourceTimeline resourceId={resourceId} />;
}
