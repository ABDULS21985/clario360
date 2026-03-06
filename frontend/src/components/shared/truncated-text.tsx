"use client";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { cn } from "@/lib/utils";

interface TruncatedTextProps {
  text: string;
  maxLength?: number;
  className?: string;
}

export function TruncatedText({ text, maxLength = 50, className }: TruncatedTextProps) {
  if (text.length <= maxLength) {
    return <span className={className}>{text}</span>;
  }

  const truncated = text.slice(0, maxLength - 3) + "...";

  return (
    <HoverCard>
      <HoverCardTrigger asChild>
        <span className={cn("cursor-default", className)} title={text}>
          {truncated}
        </span>
      </HoverCardTrigger>
      <HoverCardContent className="w-80 text-sm" align="start">
        {text}
      </HoverCardContent>
    </HoverCard>
  );
}
