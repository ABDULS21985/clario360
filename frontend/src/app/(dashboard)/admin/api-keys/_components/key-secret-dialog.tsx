"use client";

import { AlertTriangle, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CopyButton } from "@/components/shared/copy-button";

interface KeySecretDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  secret: string;
}

export function KeySecretDialog({ open, onOpenChange, secret }: KeySecretDialogProps) {
  if (!secret) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CheckCircle className="h-5 w-5 text-green-600" />
            API Key Created
          </DialogTitle>
          <DialogDescription>
            Copy your API key now. You won&apos;t be able to see it again.
          </DialogDescription>
        </DialogHeader>

        <div className="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-900/20 p-4 space-y-3">
          <div className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <p className="text-sm font-medium">
              This secret will not be shown again
            </p>
          </div>
          <div className="flex items-center gap-2">
            <code className="flex-1 text-xs font-mono bg-white dark:bg-gray-900 border rounded px-3 py-2 overflow-auto select-all break-all">
              {secret}
            </code>
            <CopyButton value={secret} label="Copy API key" size="md" />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
