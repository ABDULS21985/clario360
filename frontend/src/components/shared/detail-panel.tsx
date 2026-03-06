"use client";
import { X } from "lucide-react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";

interface DetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
  width?: "sm" | "md" | "lg" | "xl";
}

const widthClasses = {
  sm: "sm:max-w-sm",
  md: "sm:max-w-md",
  lg: "sm:max-w-lg",
  xl: "sm:max-w-xl",
};

export function DetailPanel({
  open,
  onOpenChange,
  title,
  description,
  children,
  className,
  width = "lg",
}: DetailPanelProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("p-0 flex flex-col", widthClasses[width], className)}>
        <SheetHeader className="px-6 py-4 border-b border-border shrink-0">
          <SheetTitle>{title}</SheetTitle>
          {description && <SheetDescription>{description}</SheetDescription>}
        </SheetHeader>
        <ScrollArea className="flex-1">
          <div className="px-6 py-4">{children}</div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
