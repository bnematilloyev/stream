"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/authStore";
import { Header } from "@/components/layout/Header";
import { AdminMobileNav, AdminSidebar } from "@/components/admin/AdminNav";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const hydrated = useAuthStore((s) => s.hydrated);

  useEffect(() => {
    if (!hydrated) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (user.role !== "admin") {
      router.replace("/");
    }
  }, [hydrated, user, router]);

  if (!hydrated || !user || user.role !== "admin") return null;

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="mx-auto max-w-[1400px] px-4 py-4 sm:py-6 lg:px-6">
        <div className="flex gap-6">
          <AdminSidebar />
          <main className="min-w-0 flex-1">
            <AdminMobileNav />
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
