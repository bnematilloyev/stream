"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/authStore";
import { Header } from "@/components/layout/Header";
import { StudioMobileNav, StudioSidebar } from "@/components/studio/StudioNav";

export default function StudioLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const hydrated = useAuthStore((s) => s.hydrated);

  useEffect(() => {
    if (hydrated && !user) router.replace("/login");
  }, [hydrated, user, router]);

  if (!hydrated || !user) return null;

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="mx-auto max-w-[1400px] px-4 py-4 sm:py-6 lg:px-6">
        <div className="flex gap-6">
          <StudioSidebar />
          <main className="min-w-0 flex-1">
            <StudioMobileNav />
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
