"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  FileText,
  LayoutDashboard,
  Radio,
  Shield,
  Users,
} from "lucide-react";
import { cn } from "@/lib/utils";

export const adminLinks = [
  { href: "/admin", label: "Dashboard", icon: LayoutDashboard },
  { href: "/admin/users", label: "Foydalanuvchilar", icon: Users },
  { href: "/admin/channels", label: "Kanallar", icon: Radio },
  { href: "/admin/streams", label: "Jonli efirlar", icon: Activity },
  { href: "/admin/audit", label: "Audit log", icon: FileText },
] as const;

export function AdminSidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden w-56 shrink-0 lg:block">
      <div className="mb-4 flex items-center gap-2 px-3 text-sm font-semibold text-brand">
        <Shield className="h-4 w-4" />
        Admin Panel
      </div>
      <nav className="space-y-1">
        {adminLinks.map((link) => {
          const Icon = link.icon;
          const active =
            link.href === "/admin"
              ? pathname === "/admin"
              : pathname.startsWith(link.href);
          return (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
                active
                  ? "bg-brand-light text-brand"
                  : "text-muted hover:bg-surface-2 hover:text-foreground",
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {link.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}

export function AdminMobileNav() {
  const pathname = usePathname();

  return (
    <nav className="lg:hidden -mx-4 mb-4 border-b border-border bg-surface-1 px-4">
      <div className="flex gap-1 overflow-x-auto pb-px scrollbar-none">
        {adminLinks.map((link) => {
          const Icon = link.icon;
          const active =
            link.href === "/admin"
              ? pathname === "/admin"
              : pathname.startsWith(link.href);
          return (
            <Link
              key={link.href}
              href={link.href}
              className={cn(
                "flex shrink-0 items-center gap-2 whitespace-nowrap rounded-t-xl border-b-2 px-3 py-3 text-sm font-medium transition-colors",
                active
                  ? "border-brand text-brand"
                  : "border-transparent text-muted hover:text-foreground",
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {link.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
