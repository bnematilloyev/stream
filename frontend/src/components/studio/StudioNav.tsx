"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Clapperboard, Camera, Radio, Settings } from "lucide-react";
import { cn } from "@/lib/utils";

export const studioLinks = [
  { href: "/studio", label: "Dashboard", icon: Clapperboard },
  { href: "/studio/broadcast", label: "Kamera efir", icon: Camera },
  { href: "/studio/stream", label: "OBS / RTMP", icon: Radio },
  { href: "/settings", label: "Sozlamalar", icon: Settings },
] as const;

export function StudioSidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden w-56 shrink-0 lg:block">
      <nav className="space-y-1">
        {studioLinks.map((link) => {
          const Icon = link.icon;
          const active = pathname === link.href;
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

export function StudioMobileNav() {
  const pathname = usePathname();

  return (
    <nav className="lg:hidden -mx-4 mb-4 border-b border-border bg-surface-1 px-4">
      <div className="flex gap-1 overflow-x-auto pb-px scrollbar-none">
        {studioLinks.map((link) => {
          const Icon = link.icon;
          const active = pathname === link.href;
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
