'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Pencil, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Switch } from '@/components/ui/switch';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import {
  createMonitoredBrand,
  deleteMonitoredBrand,
  fetchMonitoredBrands,
  updateMonitoredBrand,
} from '@/lib/cti-api';
import { buildTagInputValue, parseTagInput } from '@/lib/cti-utils';
import type { CTIMonitoredBrand } from '@/types/cti';

interface MonitoredBrandsManagerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated?: () => void;
}

export function MonitoredBrandsManager({
  open,
  onOpenChange,
  onUpdated,
}: MonitoredBrandsManagerProps) {
  const queryClient = useQueryClient();
  const [editingBrand, setEditingBrand] = useState<CTIMonitoredBrand | null>(null);
  const [brandName, setBrandName] = useState('');
  const [domainPattern, setDomainPattern] = useState('');
  const [keywordsInput, setKeywordsInput] = useState('');
  const [deleteCandidate, setDeleteCandidate] = useState<CTIMonitoredBrand | null>(null);

  const brandsQuery = useQuery({
    queryKey: ['cti-brands'],
    queryFn: fetchMonitoredBrands,
    enabled: open,
  });

  const resetForm = () => {
    setEditingBrand(null);
    setBrandName('');
    setDomainPattern('');
    setKeywordsInput('');
  };

  useEffect(() => {
    if (!open) {
      resetForm();
    }
  }, [open]);

  const refreshBrands = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['cti-brands'] }),
      queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse'] }),
    ]);
    onUpdated?.();
  };

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!brandName.trim()) {
        throw new Error('Brand name is required');
      }

      const payload = {
        brand_name: brandName.trim(),
        domain_pattern: domainPattern.trim() || undefined,
        keywords: parseTagInput(keywordsInput),
      };

      if (editingBrand) {
        await updateMonitoredBrand(editingBrand.id, payload);
        return editingBrand;
      }

      return createMonitoredBrand(payload);
    },
    onSuccess: async () => {
      await refreshBrands();
      toast.success(editingBrand ? 'Monitored brand updated' : 'Monitored brand created');
      resetForm();
    },
    onError: () => {
      toast.error(editingBrand ? 'Failed to update monitored brand' : 'Failed to create monitored brand');
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ brand, isActive }: { brand: CTIMonitoredBrand; isActive: boolean }) => {
      await updateMonitoredBrand(brand.id, { is_active: isActive });
    },
    onSuccess: async () => {
      await refreshBrands();
      toast.success('Brand status updated');
    },
    onError: () => {
      toast.error('Failed to update brand status');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (brandId: string) => deleteMonitoredBrand(brandId),
    onSuccess: async () => {
      await refreshBrands();
      toast.success('Monitored brand deleted');
      setDeleteCandidate(null);
      if (editingBrand?.id === deleteCandidate?.id) {
        resetForm();
      }
    },
    onError: () => {
      toast.error('Failed to delete monitored brand');
    },
  });

  const handleEdit = (brand: CTIMonitoredBrand) => {
    setEditingBrand(brand);
    setBrandName(brand.brand_name);
    setDomainPattern(brand.domain_pattern ?? '');
    setKeywordsInput(buildTagInputValue(brand.keywords));
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="w-[min(100vw-1rem,42rem)] sm:max-w-[42rem]">
          <SheetHeader>
            <SheetTitle>Monitored Brands</SheetTitle>
            <SheetDescription>
              Maintain the brand catalogue used by brand-abuse monitoring, triage, and takedown workflows.
            </SheetDescription>
          </SheetHeader>

          <div className="mt-6 space-y-6">
            <section className="space-y-4 rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold text-slate-950">
                    {editingBrand ? 'Edit Brand' : 'Add Brand'}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    Domain patterns and keywords help prioritize abuse reports against the correct brand.
                  </p>
                </div>
                {editingBrand && (
                  <Button type="button" variant="ghost" size="sm" onClick={resetForm}>
                    Cancel Edit
                  </Button>
                )}
              </div>

              <div className="grid gap-4">
                <div className="space-y-1.5">
                  <label htmlFor="brand_name" className="text-sm font-medium">Brand Name</label>
                  <Input id="brand_name" value={brandName} onChange={(event) => setBrandName(event.target.value)} />
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="domain_pattern" className="text-sm font-medium">Domain Pattern</label>
                  <Input
                    id="domain_pattern"
                    placeholder="*.clario360.com"
                    value={domainPattern}
                    onChange={(event) => setDomainPattern(event.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="keywords_input" className="text-sm font-medium">Keywords</label>
                  <Input
                    id="keywords_input"
                    placeholder="clario, secure portal, support"
                    value={keywordsInput}
                    onChange={(event) => setKeywordsInput(event.target.value)}
                  />
                </div>
              </div>

              <div className="flex justify-end">
                <Button type="button" onClick={() => saveMutation.mutate()} disabled={saveMutation.isPending}>
                  {saveMutation.isPending ? 'Saving...' : editingBrand ? 'Save Brand' : 'Add Brand'}
                </Button>
              </div>
            </section>

            <section className="space-y-4 rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-slate-950">Brand Catalogue</h3>
                  <p className="text-sm text-muted-foreground">
                    Toggle coverage or update keywords without leaving the current CTI workflow.
                  </p>
                </div>
                <Badge variant="outline">{brandsQuery.data?.length ?? 0} brands</Badge>
              </div>

              <ScrollArea className="max-h-[48vh] pr-3">
                <div className="space-y-3">
                  {brandsQuery.data?.length ? brandsQuery.data.map((brand) => (
                    <div key={brand.id} className="rounded-2xl border bg-background p-4">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-2">
                          <div className="flex items-center gap-2">
                            <p className="font-medium text-foreground">{brand.brand_name}</p>
                            <Badge variant={brand.is_active ? 'default' : 'outline'}>
                              {brand.is_active ? 'Active' : 'Inactive'}
                            </Badge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {brand.domain_pattern || 'No domain pattern configured'}
                          </p>
                          <div className="flex flex-wrap gap-2">
                            {brand.keywords.length ? brand.keywords.map((keyword) => (
                              <Badge key={keyword} variant="outline">{keyword}</Badge>
                            )) : (
                              <span className="text-sm text-muted-foreground">No keywords</span>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm">
                            <span className="text-muted-foreground">Active</span>
                            <Switch
                              checked={brand.is_active}
                              onCheckedChange={(checked) => toggleMutation.mutate({ brand, isActive: checked })}
                              aria-label={`Toggle ${brand.brand_name}`}
                            />
                          </div>
                          <Button type="button" variant="outline" size="icon" onClick={() => handleEdit(brand)}>
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button type="button" variant="outline" size="icon" onClick={() => setDeleteCandidate(brand)}>
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </div>
                    </div>
                  )) : (
                    <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                      {brandsQuery.isLoading ? 'Loading monitored brands…' : 'No monitored brands configured yet.'}
                    </div>
                  )}
                </div>
              </ScrollArea>
            </section>
          </div>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={Boolean(deleteCandidate)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setDeleteCandidate(null);
          }
        }}
        title="Delete monitored brand"
        description="This removes the brand from the catalogue and can break existing brand-abuse triage filters."
        confirmLabel="Delete Brand"
        variant="destructive"
        typeToConfirm={deleteCandidate?.brand_name}
        loading={deleteMutation.isPending}
        onConfirm={async () => {
          if (!deleteCandidate) {
            return;
          }
          await deleteMutation.mutateAsync(deleteCandidate.id);
        }}
      />
    </>
  );
}