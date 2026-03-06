'use client';

import { useSidebarStore } from '@/stores/sidebar-store';

export function useSidebar() {
  const collapsed = useSidebarStore((s) => s.collapsed);
  const mobileOpen = useSidebarStore((s) => s.mobileOpen);
  const setCollapsed = useSidebarStore((s) => s.setCollapsed);
  const toggleCollapsed = useSidebarStore((s) => s.toggleCollapsed);
  const setMobileOpen = useSidebarStore((s) => s.setMobileOpen);
  const toggleMobileOpen = useSidebarStore((s) => s.toggleMobileOpen);

  return {
    collapsed,
    mobileOpen,
    setCollapsed,
    toggleCollapsed,
    setMobileOpen,
    toggleMobileOpen,
  };
}
