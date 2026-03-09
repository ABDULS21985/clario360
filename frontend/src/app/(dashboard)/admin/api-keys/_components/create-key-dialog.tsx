"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { CalendarIcon } from "lucide-react";
import { format } from "date-fns";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Calendar } from "@/components/ui/calendar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useCreateApiKey } from "@/hooks/use-api-keys";
import { cn } from "@/lib/utils";
import { API_KEY_SCOPE_GROUPS, type ApiKeyScope } from "@/types/api-key";

const createKeySchema = z.object({
  name: z.string().min(1, "Name is required").max(100),
});

type CreateKeyFormData = z.infer<typeof createKeySchema>;

interface CreateKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (secret: string) => void;
}

export function CreateKeyDialog({ open, onOpenChange, onCreated }: CreateKeyDialogProps) {
  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [selectedScopes, setSelectedScopes] = useState<ApiKeyScope[]>([]);
  const [expiryDate, setExpiryDate] = useState<Date | undefined>(undefined);
  const [neverExpires, setNeverExpires] = useState(true);
  const createMutation = useCreateApiKey();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    getValues,
  } = useForm<CreateKeyFormData>({
    resolver: zodResolver(createKeySchema),
  });

  const handleClose = () => {
    setStep(1);
    setSelectedScopes([]);
    setExpiryDate(undefined);
    setNeverExpires(true);
    reset();
    onOpenChange(false);
  };

  const toggleScope = (scope: ApiKeyScope) => {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
  };

  const toggleGroup = (scopes: ApiKeyScope[]) => {
    const allSelected = scopes.every((s) => selectedScopes.includes(s));
    if (allSelected) {
      setSelectedScopes((prev) => prev.filter((s) => !scopes.includes(s)));
    } else {
      setSelectedScopes((prev) => [...new Set([...prev, ...scopes])]);
    }
  };

  const handleCreate = async () => {
    const name = getValues("name");
    const result = await createMutation.mutateAsync({
      name,
      scopes: selectedScopes,
      expires_at: neverExpires || !expiryDate ? null : expiryDate.toISOString(),
    });
    handleClose();
    onCreated(result.secret);
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && handleClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Create API Key</DialogTitle>
          <DialogDescription>
            {step === 1 && "Name your API key"}
            {step === 2 && "Select permissions for this key"}
            {step === 3 && "Set an expiration date (optional)"}
          </DialogDescription>
        </DialogHeader>

        {/* Step indicators */}
        <div className="flex items-center gap-2 mb-4">
          {[1, 2, 3].map((s) => (
            <div
              key={s}
              className={cn(
                "h-1.5 flex-1 rounded-full transition-colors",
                s <= step ? "bg-primary" : "bg-muted",
              )}
            />
          ))}
        </div>

        {step === 1 && (
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="key-name">Key Name</Label>
              <Input
                id="key-name"
                {...register("name")}
                placeholder="e.g. CI/CD Pipeline, Mobile App"
                autoFocus
              />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4 max-h-[400px] overflow-y-auto">
            {API_KEY_SCOPE_GROUPS.map((group) => {
              const allSelected = group.scopes.every((s) => selectedScopes.includes(s));
              return (
                <div key={group.label} className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id={`group-${group.label}`}
                      checked={allSelected}
                      onCheckedChange={() => toggleGroup(group.scopes)}
                    />
                    <Label
                      htmlFor={`group-${group.label}`}
                      className="font-medium text-sm cursor-pointer"
                    >
                      {group.label}
                    </Label>
                  </div>
                  <div className="ml-6 grid grid-cols-2 gap-2">
                    {group.scopes.map((scope) => (
                      <div key={scope} className="flex items-center gap-2">
                        <Checkbox
                          id={`scope-${scope}`}
                          checked={selectedScopes.includes(scope)}
                          onCheckedChange={() => toggleScope(scope)}
                        />
                        <Label
                          htmlFor={`scope-${scope}`}
                          className="text-xs font-mono cursor-pointer"
                        >
                          {scope}
                        </Label>
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
            {selectedScopes.length === 0 && (
              <p className="text-xs text-destructive">Select at least one scope</p>
            )}
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Checkbox
                id="never-expires"
                checked={neverExpires}
                onCheckedChange={(checked) => {
                  setNeverExpires(!!checked);
                  if (checked) setExpiryDate(undefined);
                }}
              />
              <Label htmlFor="never-expires" className="cursor-pointer">
                Never expires
              </Label>
            </div>

            {!neverExpires && (
              <div className="space-y-1.5">
                <Label>Expiration Date</Label>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                      variant="outline"
                      className={cn(
                        "w-full justify-start text-left font-normal",
                        !expiryDate && "text-muted-foreground",
                      )}
                    >
                      <CalendarIcon className="mr-2 h-4 w-4" />
                      {expiryDate ? format(expiryDate, "PPP") : "Pick a date"}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      mode="single"
                      selected={expiryDate}
                      onSelect={setExpiryDate}
                      disabled={(date) => date < new Date()}
                      initialFocus
                    />
                  </PopoverContent>
                </Popover>
              </div>
            )}

            {/* Summary */}
            <div className="rounded-lg border p-3 space-y-2 text-sm">
              <p className="font-medium">Summary</p>
              <p className="text-muted-foreground">Name: {getValues("name")}</p>
              <p className="text-muted-foreground">
                Scopes: {selectedScopes.length} selected
              </p>
              <p className="text-muted-foreground">
                Expires: {neverExpires ? "Never" : expiryDate ? format(expiryDate, "PPP") : "Not set"}
              </p>
            </div>
          </div>
        )}

        <DialogFooter className="gap-2">
          {step > 1 && (
            <Button
              type="button"
              variant="outline"
              onClick={() => setStep((s) => (s - 1) as 1 | 2 | 3)}
            >
              Back
            </Button>
          )}
          {step < 3 ? (
            <Button
              type="button"
              onClick={async () => {
                if (step === 1) {
                  const valid = await handleSubmit(() => {})();
                  if (!errors.name) setStep(2);
                } else if (step === 2 && selectedScopes.length > 0) {
                  setStep(3);
                }
              }}
              disabled={step === 2 && selectedScopes.length === 0}
            >
              Next
            </Button>
          ) : (
            <Button
              type="button"
              onClick={handleCreate}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create Key"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
