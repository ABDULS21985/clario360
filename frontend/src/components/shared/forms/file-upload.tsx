"use client";
import { useRef, useState } from "react";
import { Upload, X, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";
import { formatBytes } from "@/lib/format";

interface FileUploadProps {
  accept?: string;
  maxSizeMB?: number;
  multiple?: boolean;
  onUpload: (files: File[]) => Promise<void>;
  uploading?: boolean;
  progress?: number;
  disabled?: boolean;
  className?: string;
}

export function FileUpload({
  accept,
  maxSizeMB = 100,
  multiple = false,
  onUpload,
  uploading = false,
  progress = 0,
  disabled = false,
  className,
}: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);

  const maxBytes = maxSizeMB * 1024 * 1024;

  const validate = (files: File[]): string | null => {
    for (const file of files) {
      if (file.size > maxBytes) return `"${file.name}" exceeds the maximum size of ${maxSizeMB}MB.`;
    }
    return null;
  };

  const handleFiles = async (files: File[]) => {
    setError(null);
    const validationError = validate(files);
    if (validationError) { setError(validationError); return; }
    setPendingFiles(files);
    await onUpload(files);
    if (!uploading) setPendingFiles([]);
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    if (disabled || uploading) return;
    const files = Array.from(e.dataTransfer.files);
    handleFiles(multiple ? files : [files[0]]);
  };

  return (
    <div className={cn("space-y-2", className)}>
      <div
        className={cn(
          "relative flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-8 text-center transition-colors",
          isDragging ? "border-primary bg-primary/5" : "border-muted-foreground/25 hover:border-muted-foreground/50",
          (disabled || uploading) && "opacity-50 cursor-not-allowed",
          !disabled && !uploading && "cursor-pointer"
        )}
        onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={onDrop}
        onClick={() => !disabled && !uploading && inputRef.current?.click()}
        role="button"
        tabIndex={disabled ? -1 : 0}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); inputRef.current?.click(); } }}
        aria-label="Upload file area"
      >
        <Upload className="h-8 w-8 text-muted-foreground" aria-hidden />
        <div>
          <p className="text-sm font-medium">Drag &amp; drop files here, or click to browse</p>
          <p className="text-xs text-muted-foreground mt-1">Maximum file size: {maxSizeMB}MB</p>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept={accept}
          multiple={multiple}
          className="hidden"
          onChange={(e) => {
            const files = Array.from(e.target.files ?? []);
            if (files.length > 0) handleFiles(files);
            e.target.value = "";
          }}
          disabled={disabled || uploading}
          aria-hidden
        />
      </div>

      {uploading && pendingFiles.length > 0 && (
        <div className="space-y-2">
          {pendingFiles.map((f) => (
            <div key={f.name} className="rounded border border-border p-3 space-y-1.5">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium truncate">{f.name}</p>
                <span className="text-xs text-muted-foreground ml-2 shrink-0">{formatBytes(f.size)}</span>
              </div>
              <Progress value={progress} className="h-1.5" />
            </div>
          ))}
        </div>
      )}

      {error && (
        <div className="flex items-center gap-2 text-sm text-destructive" role="alert">
          <AlertCircle className="h-4 w-4 shrink-0" aria-hidden />
          {error}
        </div>
      )}
    </div>
  );
}
