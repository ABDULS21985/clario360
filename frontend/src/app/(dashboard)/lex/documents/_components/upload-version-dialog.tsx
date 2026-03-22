'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { LexDocument } from '@/types/suites';

interface UploadVersionDialogProps {
  document: LexDocument | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UploadVersionDialog({ document, open, onOpenChange }: UploadVersionDialogProps) {
  const queryClient = useQueryClient();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [extractedText, setExtractedText] = useState('');
  const [changeSummary, setChangeSummary] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);

  function reset() {
    setSelectedFile(null);
    setExtractedText('');
    setChangeSummary('');
    setUploadProgress(0);
  }

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!document || !selectedFile) throw new Error('No file selected');

      const uploaded = await enterpriseApi.files.upload(
        selectedFile,
        {
          suite: 'lex',
          entity_type: 'document_version',
          tags: ['document', document.type].join(','),
          lifecycle_policy: 'standard',
        },
        (progress) => setUploadProgress(progress),
      );

      return enterpriseApi.lex.uploadDocumentVersion(document.id, {
        file_id: uploaded.id,
        file_name: uploaded.original_name,
        file_size_bytes: uploaded.size_bytes,
        content_hash: uploaded.checksum_sha256,
        extracted_text: extractedText.trim(),
        change_summary: changeSummary.trim(),
      });
    },
    onSuccess: async () => {
      showSuccess('Version uploaded.', 'A new document version has been added.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-documents'] }),
        document
          ? queryClient.invalidateQueries({ queryKey: ['lex-document', document.id] })
          : Promise.resolve(),
      ]);
      onOpenChange(false);
      reset();
    },
    onError: showApiError,
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) reset();
        onOpenChange(isOpen);
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Upload New Version</DialogTitle>
          <DialogDescription>
            {document
              ? `Attach a new version of "${document.title}".`
              : 'Upload a new document version.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="version-file">
              Document file <span className="text-destructive">*</span>
            </Label>
            <Input
              id="version-file"
              type="file"
              accept=".pdf,.doc,.docx,.txt,.xlsx,.pptx"
              onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
            />
            {selectedFile ? (
              <p className="text-xs text-muted-foreground">Selected: {selectedFile.name}</p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="version-summary">Change summary</Label>
            <Input
              id="version-summary"
              value={changeSummary}
              onChange={(e) => setChangeSummary(e.target.value)}
              placeholder="What changed in this version?"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="version-text">Extracted text</Label>
            <Textarea
              id="version-text"
              value={extractedText}
              onChange={(e) => setExtractedText(e.target.value)}
              placeholder="Paste document text for indexing."
              rows={4}
            />
          </div>

          {uploadMutation.isPending && selectedFile ? (
            <p className="text-xs text-muted-foreground">
              Upload progress: {Math.round(uploadProgress)}%
            </p>
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!selectedFile || uploadMutation.isPending}
            onClick={() => uploadMutation.mutate()}
          >
            {uploadMutation.isPending ? (
              <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
            ) : null}
            Upload version
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
