"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";
import {
  onAccessTokenRefreshed,
  onAuthCleared,
  setAccessToken,
} from "@/lib/api/client";
import type { User } from "@/types";

interface AuthState {
  user: User | null;
  accessToken: string | null;
  hydrated: boolean;
  setAuth: (user: User, accessToken: string) => void;
  setAccessTokenOnly: (accessToken: string) => void;
  setUser: (user: User) => void;
  clearAuth: () => void;
  setHydrated: (v: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      hydrated: false,
      setAuth: (user, accessToken) => {
        setAccessToken(accessToken);
        set({ user, accessToken });
      },
      setAccessTokenOnly: (accessToken) => {
        setAccessToken(accessToken);
        set({ accessToken });
      },
      setUser: (user) => set({ user }),
      clearAuth: () => {
        setAccessToken(null);
        set({ user: null, accessToken: null });
      },
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "sahiy-auth",
      // Faqat user saqlanadi — access token memory + refresh cookie orqali tiklanadi.
      partialize: (s) => ({ user: s.user }),
      onRehydrateStorage: () => (state) => {
        state?.setHydrated(true);
      },
    },
  ),
);

if (typeof window !== "undefined") {
  onAccessTokenRefreshed((token) => {
    useAuthStore.getState().setAccessTokenOnly(token);
  });
  onAuthCleared(() => {
    useAuthStore.getState().clearAuth();
  });
}
