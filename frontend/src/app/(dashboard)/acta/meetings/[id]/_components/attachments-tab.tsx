'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { FileUpload } from '@/components/shared/forms/file-upload';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError } from '@/lib/toast';
import type { ActaMeetingAttachment } from '@/types/suites';

interface AttachmentsTabProps {
  meetingId: string;
  attachments: ActaMeetingAttachment[];
  currentUserId?: string | null;
  onRefresh: () => Promise<void>;
}

export function AttachmentsTab({
  meetingId,
  attachments,
  currentUserId,
  onRefresh,
}: AttachmentsTabProps) {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);

  const handleUpload = async (files: File[]) => {
    setUploading(true);
    try {
      for (const file of files) {
        const uploaded = await enterpriseApi.files.upload(
          file,
          { suite: 'acta', entity_type: 'meeting_attachment', entity_id: meetingId },
          setProgress,
        );
        await enterpriseApi.acta.addAttachmentReference(meetingId, {
          file_id: uploaded.id,
          name: uploaded.original_name,
          content_type: uploaded.content_type,
          uploaded_by: currentUserId ?? null,
        });
      }
      await onRefresh();
    } catch (error) {
      showApiError(error);
    } finally {
      setUploading(false);
      setProgress(0);
    }
  };

  return (
    <div className="space-y-4">
      <FileUpload onUpload={handleUpload} uploading={uploading} progress={progress} multiple />
      <div className="space-y-3">
        {attachments.length === 0 ? (
          <p className="text-sm text-muted-foreground">No attachments have been uploaded for this meeting.</p>
        ) : (
          attachments.map((attachment) => (
            <div key={attachment.file_id} className="flex items-center justify-between rounded-xl border bg-card px-4 py-3">
              <div>
                <p className="font-medium">{attachment.name}</p>
                <p className="text-xs text-muted-foreground">
                  {attachment.content_type ?? 'Unknown type'} • {attachment.uploaded_at}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={async () => {
                  try {
                    await enterpriseApi.acta.deleteAttachment(meetingId, attachment.file_id);
                    await onRefresh();
                  } catch (error) {
                    showApiError(error);
                  }
                }}
              >
                Remove
              </Button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
