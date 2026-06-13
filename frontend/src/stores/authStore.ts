"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";
import { setAccessToken } from "@/lib/api/client";
import type { User } from "@/types";

interface AuthState {
  user: User | null;
  accessToken: string | null;
  hydrated: boolean;
  setAuth: (user: User, accessToken: string) => void;
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
      setUser: (user) => set({ user }),
      clearAuth: () => {
        setAccessToken(null);
        set({ user: null, accessToken: null });
      },
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "sahiy-auth",
      partialize: (s) => ({ user: s.user, accessToken: s.accessToken }),
      onRehydrateStorage: () => (state) => {
        if (state?.accessToken) setAccessToken(state.accessToken);
        state?.setHydrated(true);
      },
    },
  ),
);
