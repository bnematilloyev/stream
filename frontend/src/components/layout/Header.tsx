"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Radio, Search, User, LogOut, Clapperboard } from "lucide-react";
import { useAuthStore } from "@/stores/authStore";
import { logout } from "@/lib/api/auth";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/", label: "Bosh sahifa" },
  { href: "/browse", label: "Jonli" },
];

export function Header() {
  const pathname = usePathname();
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const clearAuth = useAuthStore((s) => s.clearAuth);

  async function handleLogout() {
    try {
      await logout();
    } finally {
      clearAuth();
      router.push("/");
    }
  }

  return (
    <header className="brand-bg sticky top-0 z-50 border-b border-white/10 shadow-md">
      <div className="mx-auto flex h-16 max-w-[1600px] items-center gap-6 px-4 lg:px-6">
        <Link href="/" className="flex items-center gap-2 shrink-0">
          <div className="icon-gold-gradient flex h-9 w-9 items-center justify-center rounded-xl shadow-lg">
            <Radio className="h-5 w-5 text-brand" />
          </div>
          <span className="hidden text-lg font-bold tracking-tight text-white sm:block">
            Sahiy<span className="text-gradient-gold">Stream</span>
          </span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {nav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                pathname === item.href
                  ? "bg-white/15 text-white"
                  : "text-white/70 hover:bg-white/10 hover:text-white",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          <Link
            href="/browse"
            className="hidden rounded-xl p-2 text-white/70 transition-colors hover:bg-white/10 hover:text-white sm:flex"
          >
            <Search className="h-5 w-5" />
          </Link>

          {user ? (
            <>
              <Link href="/studio">
                <Button variant="secondary" size="sm">
                  <Clapperboard className="h-4 w-4" />
                  <span className="hidden sm:inline">Studio</span>
                </Button>
              </Link>
              <Link
                href="/settings"
                className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-white/80 transition-colors hover:bg-white/10 hover:text-white"
              >
                <User className="h-4 w-4" />
                <span className="hidden max-w-[120px] truncate font-medium md:inline">
                  {user.display_name || user.username}
                </span>
              </Link>
              <button
                onClick={handleLogout}
                className="rounded-xl p-2 text-white/70 transition-colors hover:bg-white/10 hover:text-white"
                title="Chiqish"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </>
          ) : (
            <>
              <Link href="/login">
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-white/80 hover:bg-white/10 hover:text-white"
                >
                  Kirish
                </Button>
              </Link>
              <Link href="/register">
                <Button size="sm">Ro&apos;yxatdan o&apos;tish</Button>
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
