"use client";


import { ResourceTimeline } from "./_components/resource-timeline";

interface ResourceTimelinePageProps {
  params: { resourceId: string };
}

export default function ResourceTimelinePage({
  params,
}: ResourceTimelinePageProps) {
  const { resourceId } = params;

  return <ResourceTimeline resourceId={resourceId} />;
}
