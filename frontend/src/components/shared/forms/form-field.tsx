"use client";
import { useFormContext } from "react-hook-form";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface FormFieldProps {
  name: string;
  label: string;
  description?: string;
  required?: boolean;
  disabled?: boolean;
  children: React.ReactNode;
  className?: string;
}

export function FormField({
  name,
  label,
  description,
  required = false,
  disabled = false,
  children,
  className,
}: FormFieldProps) {
  const { formState: { errors } } = useFormContext();
  const error = errors[name];
  const errorMessage = typeof error?.message === "string" ? error.message : undefined;

  return (
    <div className={cn("space-y-1.5", className)}>
      <Label
        htmlFor={name}
        className={cn(disabled && "opacity-50", error && "text-destructive")}
      >
        {label}
        {required && <span className="text-destructive ml-0.5" aria-hidden>*</span>}
      </Label>
      <div id={name} aria-describedby={description ? `${name}-desc` : undefined} aria-invalid={!!error} aria-required={required}>
        {children}
      </div>
      {description && (
        <p id={`${name}-desc`} className="text-xs text-muted-foreground">{description}</p>
      )}
      {errorMessage && (
        <p className="text-xs text-destructive" role="alert" aria-live="polite">{errorMessage}</p>
      )}
    </div>
  );
}
