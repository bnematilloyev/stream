"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/authStore";
import { Header } from "@/components/layout/Header";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default function SettingsPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const hydrated = useAuthStore((s) => s.hydrated);

  useEffect(() => {
    if (hydrated && !user) router.replace("/login");
  }, [hydrated, user, router]);

  if (!user) return null;

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="mx-auto max-w-2xl px-4 py-8 lg:px-6">
        <h1 className="mb-6 text-2xl font-bold">Sozlamalar</h1>
        <Card>
          <CardHeader>
            <h2 className="font-semibold">Profil</h2>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-muted">Username</p>
              <p className="font-medium">@{user.username}</p>
            </div>
            <div>
              <p className="text-sm text-muted">Ko&apos;rinadigan ism</p>
              <p className="font-medium">{user.display_name}</p>
            </div>
            <div>
              <p className="text-sm text-muted">Email</p>
              <p className="font-medium">{user.email}</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
