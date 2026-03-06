"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const navigation = [
  { name: "Dashboard", href: "/", icon: "LayoutDashboard" },
  { name: "Cybersecurity", href: "/cyber", icon: "Shield" },
  { name: "Data Suite", href: "/data", icon: "Database" },
  { name: "Acta", href: "/acta", icon: "FileText" },
  { name: "Lex", href: "/lex", icon: "Scale" },
  { name: "Visus360", href: "/visus", icon: "BarChart3" },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside
      className={cn(
        "flex h-full flex-col border-r bg-card transition-all duration-300",
        collapsed ? "w-16" : "w-64"
      )}
    >
      <div className="flex h-14 items-center border-b px-4">
        {!collapsed && (
          <Link href="/" className="text-lg font-bold text-primary">
            Clario 360
          </Link>
        )}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className={cn(
            "rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-accent-foreground",
            collapsed ? "mx-auto" : "ml-auto"
          )}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {collapsed ? ">>" : "<<"}
        </button>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navigation.map((item) => {
          const isActive =
            item.href === "/"
              ? pathname === "/"
              : pathname.startsWith(item.href);

          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )}
              title={collapsed ? item.name : undefined}
            >
              <span className="flex h-5 w-5 items-center justify-center text-xs">
                {item.icon.charAt(0)}
              </span>
              {!collapsed && <span>{item.name}</span>}
            </Link>
          );
        })}
      </nav>

      <div className="border-t p-2">
        <Link
          href="/settings"
          className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          title={collapsed ? "Settings" : undefined}
        >
          <span className="flex h-5 w-5 items-center justify-center text-xs">S</span>
          {!collapsed && <span>Settings</span>}
        </Link>
      </div>
    </aside>
  );
}
