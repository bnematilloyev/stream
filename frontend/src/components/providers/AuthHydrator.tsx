"use client";

import { useEffect, useState } from "react";
import { restoreSession } from "@/lib/api/auth";
import { useAuthStore } from "@/stores/authStore";

export function AuthHydrator({ children }: { children: React.ReactNode }) {
  const hydrated = useAuthStore((s) => s.hydrated);
  const user = useAuthStore((s) => s.user);
  const [sessionReady, setSessionReady] = useState(false);

  useEffect(() => {
    if (!hydrated) return;

    if (!user) {
      setSessionReady(true);
      return;
    }

    let cancelled = false;
    restoreSession().finally(() => {
      if (!cancelled) setSessionReady(true);
    });

    return () => {
      cancelled = true;
    };
  }, [hydrated, user]);

  if (!hydrated || (user && !sessionReady)) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-brand border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
