"use client";
import { Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useClipboard } from "@/hooks/use-clipboard";
import { cn } from "@/lib/utils";

interface CopyButtonProps {
  value: string;
  label?: string;
  size?: "sm" | "md";
  className?: string;
}

export function CopyButton({ value, label = "Copy", size = "sm", className }: CopyButtonProps) {
  const { copy, copied } = useClipboard(2000);
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className={cn(size === "sm" ? "h-6 w-6" : "h-8 w-8", className)}
            onClick={(e) => { e.stopPropagation(); copy(value); }}
            aria-label={copied ? "Copied!" : label}
          >
            {copied ? (
              <Check className="h-3 w-3 text-green-600" />
            ) : (
              <Copy className={cn("text-muted-foreground", size === "sm" ? "h-3 w-3" : "h-4 w-4")} />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>{copied ? "Copied!" : label}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
