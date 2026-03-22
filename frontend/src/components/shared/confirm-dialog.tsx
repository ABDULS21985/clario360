"use client";
import { useState } from "react";
import { AlertTriangle } from "lucide-react";
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "destructive";
  typeToConfirm?: string; // user must type this exact string
  onConfirm: () => Promise<void> | void;
  loading?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "default",
  typeToConfirm,
  onConfirm,
  loading = false,
}: ConfirmDialogProps) {
  const [typedValue, setTypedValue] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const canConfirm = !typeToConfirm || typedValue === typeToConfirm;

  const handleConfirm = async () => {
    setIsSubmitting(true);
    try {
      await onConfirm();
      onOpenChange(false);
      setTypedValue("");
    } catch {
      // Caller already handles the error (toast, state); swallow here so the dialog
      // stays open for retry rather than producing an unhandled promise rejection.
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) setTypedValue("");
    onOpenChange(o);
  };

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <div className="flex items-center gap-3">
            {variant === "destructive" && (
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-destructive/10">
                <AlertTriangle className="h-5 w-5 text-destructive" aria-hidden />
              </div>
            )}
            <div>
              <AlertDialogTitle>{title}</AlertDialogTitle>
              <AlertDialogDescription className="mt-1">{description}</AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>

        {typeToConfirm && (
          <div className="space-y-2">
            <Label htmlFor="confirm-input" className="text-sm">
              Type <span className="font-mono font-bold">{typeToConfirm}</span> to confirm
            </Label>
            <Input
              id="confirm-input"
              value={typedValue}
              onChange={(e) => setTypedValue(e.target.value)}
              placeholder={typeToConfirm}
              className={cn(typedValue && typedValue !== typeToConfirm && "border-destructive")}
            />
          </div>
        )}

        <AlertDialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting || loading}>
            {cancelLabel}
          </Button>
          <Button
            variant={variant === "destructive" ? "destructive" : "default"}
            onClick={handleConfirm}
            disabled={!canConfirm || isSubmitting || loading}
          >
            {isSubmitting || loading ? "Processing..." : confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
